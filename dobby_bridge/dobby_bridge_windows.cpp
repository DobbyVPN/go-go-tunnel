#include "dobby_bridge_common.h"

#include <memory>
#include <string>
#include <thread>
#include <future>

#include <magic_enum/magic_enum.hpp>
#include <toml++/toml.h>

// TrustTunnel Internal Headers
#include "common/logger.h"
#include "common/net_utils.h"
#include "net/tls.h"
#include "vpn/event_loop.h"
#include "vpn/platform.h"
#include "vpn/trusttunnel/auto_network_monitor.h"
#include "vpn/trusttunnel/client.h"
#include "vpn/trusttunnel/config.h"

// Windows-specific headers
#include <windows.h>
#include <aclapi.h>

static ag::Logger g_logger{"DobbyBridgeWindows"};

// Global callbacks
static dobby_on_log_message_t g_log_callback = nullptr;
static dobby_on_protect_socket_t g_protect_callback = nullptr;

// Windows-specific globals
static INIT_ONCE g_init_once = INIT_ONCE_STATIC_INIT;
static HMODULE g_wintun_handle = nullptr;

// VPN context
struct dobby_vpn_context {
    std::string config_str;
    toml::parse_result parsed_config;
    std::unique_ptr<ag::TrustTunnelClient> client;
    std::unique_ptr<ag::AutoNetworkMonitor> network_monitor;
};

static dobby_vpn_context* m_vpn = nullptr;
static std::unique_ptr<ag::VpnEventLoop, decltype(&ag::vpn_event_loop_destroy)> m_ev_loop{nullptr, ag::vpn_event_loop_destroy};
static std::thread m_executor_thread;

// Windows-specific certificate verification
static void vpn_windows_verify_certificate(ag::VpnVerifyCertificateEvent *event) {
    event->result = !!ag::tls_verify_cert(event->cert, event->chain, nullptr);
}

// Windows-specific socket protection
// This function can be overridden by the user via dobby_vpn_set_protect_callback
static int vpn_windows_protect_socket(int fd) {
    // If user provided a custom callback, use it
    if (g_protect_callback) {
        return g_protect_callback(fd);
    }
    
    // Default Windows implementation: use TrustTunnel's built-in protection
    // Note: This requires the peer address which we don't have here
    // The actual protection will be done in the protect_handler below
    return 0; // Success
}

// Set log callback
void dobby_vpn_set_log_callback(dobby_on_log_message_t cb) {
    g_log_callback = cb;
    if (cb != nullptr) {
        ag::Logger::set_callback([](ag::LogLevel log_level, std::string_view message) {
            if (g_log_callback) {
                std::string msg_str(message);
                g_log_callback(static_cast<int>(log_level), msg_str.c_str());
            }
        });
    } else {
        ag::Logger::set_callback(nullptr);
    }
}

// Set socket protection callback - allows user to provide their own implementation
void dobby_vpn_set_protect_callback(dobby_on_protect_socket_t cb) {
    g_protect_callback = cb;
}

// Start VPN
void dobby_vpn_start(const char *toml_config, dobby_on_state_changed_t state_changed_cb, void *state_changed_cb_arg) {
    if (m_vpn) {
        warnlog(g_logger, "VPN is already running.");
        return;
    }

    // 1. Start Event Loop immediately
    m_ev_loop.reset(ag::vpn_event_loop_create());
    m_executor_thread = std::thread([]() { ag::vpn_event_loop_run(m_ev_loop.get()); });

    // 2. Capture the RAW string, which is safely copyable across threads
    std::string config_str(toml_config);

    ag::event_loop::submit(m_ev_loop.get(), [config_str, state_changed_cb, state_changed_cb_arg]() {
        
        m_vpn = new dobby_vpn_context();
        m_vpn->config_str = config_str;

        // 3. Parse the TOML *inside* the background thread
        m_vpn->parsed_config = toml::parse(m_vpn->config_str);
        if (!m_vpn->parsed_config) {
            errlog(g_logger, "Failed to parse TOML config");
            delete m_vpn;
            m_vpn = nullptr;
            return;
        }

        auto trusttunnel_config = ag::TrustTunnelConfig::build_config(m_vpn->parsed_config);
        if (!trusttunnel_config) {
            errlog(g_logger, "Failed to build TrustTunnel config from TOML");
            delete m_vpn;
            m_vpn = nullptr;
            return;
        }
        ag::vpn_post_quantum_group_set_enabled(trusttunnel_config->post_quantum_group_enabled);

        ag::VpnCallbacks callbacks;
        
        // Windows-specific socket protection handler
        if (std::holds_alternative<ag::TrustTunnelConfig::TunListener>(trusttunnel_config->listener)) {
            // For TUN listener, use Windows Wintun protection
            callbacks.protect_handler = [](ag::SocketProtectEvent *event) {
                // Use TrustTunnel's built-in Windows socket protection
                event->result = !ag::vpn_win_socket_protect(event->fd, event->peer);
            };
        } else {
            // For other listeners (SOCKS, etc.), use user callback or default
            callbacks.protect_handler = [](ag::SocketProtectEvent *event) {
                // If Go provides a callback, use it. Otherwise, default to 0 (Allow).
                event->result = g_protect_callback ? g_protect_callback(event->fd) : 0;
            };
        }
        
        // Windows-specific certificate verification
        callbacks.verify_handler = [](ag::VpnVerifyCertificateEvent *event) {
            vpn_windows_verify_certificate(event);
        };

        // State change handler
        callbacks.state_changed_handler = [state_changed_cb, state_changed_cb_arg](ag::VpnStateChangedEvent *event) {
            infolog(g_logger, "State changed: {}", magic_enum::enum_name(event->state));
            if (state_changed_cb) state_changed_cb(state_changed_cb_arg, event->state);
        };

        // 4. Initialize the client safely
        m_vpn->client = std::make_unique<ag::TrustTunnelClient>(std::move(*trusttunnel_config), std::move(callbacks));

        // Optional: Start network monitor
        // m_vpn->network_monitor = std::make_unique<ag::AutoNetworkMonitor>(m_vpn->client.get());
        // m_vpn->network_monitor->start();
        
        m_vpn->client->connect(ag::TrustTunnelClient::AutoSetup{});
        
    }).release();
}

// Stop VPN
void dobby_vpn_stop() {
    infolog(g_logger, "Starting TrustTunnel vpn core stop.");

    // Always stop the event loop if it exists, even if m_vpn is currently null
    if (m_ev_loop) {
        std::promise<void> stop_promise;
        std::future<void> stop_future = stop_promise.get_future();

        // Submit cleanup task
        ag::event_loop::submit(m_ev_loop.get(), [&stop_promise]() {
            if (m_vpn) {
                // Let the destructor handle all the safe async teardown!
                delete m_vpn;
                m_vpn = nullptr;
            }
            stop_promise.set_value(); 
        }).release();

        // Wait for cleanup task to finish safely
        stop_future.wait();

        // Stop the loop
        ag::vpn_event_loop_stop(m_ev_loop.get());
    }

    // Always join the thread
    if (m_executor_thread.joinable()) {
        m_executor_thread.join();
    }

    // Free the event loop memory so it is completely clean for the next start
    m_ev_loop.reset();

    infolog(g_logger, "End TrustTunnel vpn core stop.");
}
