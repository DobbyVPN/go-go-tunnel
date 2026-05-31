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

// Android-specific headers
#include <jni.h>
#include <sys/socket.h>
#include <netinet/in.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>

static ag::Logger g_logger{"DobbyBridgeAndroid"};

// Global callbacks
static dobby_on_log_message_t g_log_callback = nullptr;
static dobby_on_protect_socket_t g_protect_callback = nullptr;

// Android-specific JNI globals
static JavaVM *g_jvm = nullptr;
static jobject g_vpn_service = nullptr;
static jmethodID g_protect_method = nullptr;

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

// Android-specific socket protection
// This function can be overridden by the user via dobby_vpn_set_protect_callback
static int vpn_android_protect_socket(int fd) {
    // If user provided a custom callback, use it
    if (g_protect_callback) {
        return g_protect_callback(fd);
    }
    
    // Default Android implementation: use VpnService.protect()
    if (g_jvm && g_vpn_service && g_protect_method) {
        JNIEnv *env = nullptr;
        if (g_jvm->GetEnv(reinterpret_cast<void**>(&env), JNI_VERSION_1_6) == JNI_OK) {
            jint result = env->CallIntMethod(g_vpn_service, g_protect_method, fd);
            if (env->ExceptionCheck()) {
                env->ExceptionDescribe();
                env->ExceptionClear();
                return -1;
            }
            return result == 0 ? 0 : -1;
        }
    }
    
    // No VPN service available, return error
    return -1;
}

// Initialize Android-specific components
// Call this from Java before starting the VPN
extern "C" DOBBY_EXPORT void dobby_vpn_android_init(JNIEnv *env, jobject vpn_service) {
    if (g_vpn_service) {
        env->DeleteGlobalRef(g_vpn_service);
    }
    
    g_vpn_service = env->NewGlobalRef(vpn_service);
    
    // Get the VpnService class
    jclass vpn_service_class = env->GetObjectClass(vpn_service);
    if (vpn_service_class) {
        g_protect_method = env->GetMethodID(vpn_service_class, "protect", "(I)I");
    }
    
    env->GetJavaVM(&g_jvm);
}

// Cleanup Android-specific components
extern "C" DOBBY_EXPORT void dobby_vpn_android_cleanup(JNIEnv *env) {
    if (g_vpn_service) {
        env->DeleteGlobalRef(g_vpn_service);
        g_vpn_service = nullptr;
    }
    g_protect_method = nullptr;
    g_jvm = nullptr;
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
        ag::vpn_post_quantum_group_set_enabled(trusttunnel_config->post_quantum_group_enabled);

        ag::VpnCallbacks callbacks;
        
        // Android-specific socket protection handler
        callbacks.protect_handler = [](ag::SocketProtectEvent *event) {
            // If Go provides a callback, use it. Otherwise, use Android VpnService.protect()
            if (g_protect_callback) {
                event->result = g_protect_callback(event->fd);
            } else {
                event->result = vpn_android_protect_socket(event->fd);
            }
        };
        
        // Android-specific certificate verification
        callbacks.verify_handler = [](ag::VpnVerifyCertificateEvent *event) {
            // Use TrustTunnel's default certificate verification
            event->result = !!ag::tls_verify_cert(event->cert, event->chain, nullptr);
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
