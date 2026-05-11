#import "dobby_bridge_common.h"

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

// iOS-specific headers
#include <net/if.h>
#include <netinet/in.h>
#include <ifaddrs.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <unistd.h>
#include <errno.h>
#include <string.h>

static ag::Logger g_logger{"DobbyBridgeIOS"};

// Global callbacks
static dobby_on_log_message_t g_log_callback = nullptr;
static dobby_on_protect_socket_t g_protect_callback = nullptr;

// iOS-specific globals
static uint32_t g_outbound_interface = 0;

// VPN context
struct dobby_vpn_context {
    std::unique_ptr<ag::TrustTunnelClient> client;
    std::unique_ptr<ag::AutoNetworkMonitor> network_monitor;
};

static dobby_vpn_context* m_vpn = nullptr;
static std::unique_ptr<ag::VpnEventLoop, decltype(&ag::vpn_event_loop_destroy)> m_ev_loop{nullptr, ag::vpn_event_loop_destroy};
static std::thread m_executor_thread;

// Helper function to get interface address
static struct sockaddr_in get_interface_address(const char *if_name, int family) {
    struct sockaddr_in addr = {};
    addr.sin_family = family;
    
    struct ifaddrs *ifaddr, *ifa;
    if (getifaddrs(&ifaddr) == 0) {
        for (ifa = ifaddr; ifa != NULL; ifa = ifa->ifa_next) {
            if (0 != strcmp(ifa->ifa_name, if_name)) {
                continue;
            }
            if (ifa->ifa_addr->sa_family == family) {
                if (family == AF_INET6) {
                    struct sockaddr_in6 *sin6 = (struct sockaddr_in6 *)ifa->ifa_addr;
                    // Skip link-local addresses
                    if (sin6->sin6_addr.s6_addr[0] == 0xfe && sin6->sin6_addr.s6_addr[1] == 0x80) {
                        continue;
                    }
                }
                memcpy(&addr, ifa->ifa_addr, ifa->ifa_addr->sa_len);
                break;
            }
        }
        freeifaddrs(ifaddr);
    }
    return addr;
}

// iOS-specific socket protection
// This function can be overridden by the user via dobby_vpn_set_protect_callback
static int vpn_ios_protect_socket(int fd, const struct sockaddr *peer) {
    // If user provided a custom callback, use it
    if (g_protect_callback) {
        return g_protect_callback(fd);
    }
    
    // Default iOS implementation: bind to outbound interface
    if (g_outbound_interface == 0) {
        return -1; // No interface set
    }
    
    char if_name[IF_NAMESIZE] = "not set";
    if_indextoname(g_outbound_interface, if_name);
    
    if (peer->sa_family == AF_INET) {
        // Set IP_BOUND_IF socket option
        if (setsockopt(fd, IPPROTO_IP, IP_BOUND_IF, &g_outbound_interface, sizeof(g_outbound_interface)) != 0) {
            errlog(g_logger, "Setsockopt IP_BOUND_IF failed: {}", strerror(errno));
            return -1;
        }
        
        // Bind to interface address
        struct sockaddr_in addr = get_interface_address(if_name, AF_INET);
        if (addr.sin_family == AF_INET) {
            if (bind(fd, (struct sockaddr *)&addr, sizeof(addr)) != 0) {
                errlog(g_logger, "Bind to {} failed: {}", if_name, strerror(errno));
                return -1;
            }
        }
    } else if (peer->sa_family == AF_INET6) {
        // Set IPV6_BOUND_IF socket option
        if (setsockopt(fd, IPPROTO_IPV6, IPV6_BOUND_IF, &g_outbound_interface, sizeof(g_outbound_interface)) != 0) {
            errlog(g_logger, "Setsockopt IPV6_BOUND_IF failed: {}", strerror(errno));
            return -1;
        }
        
        // Bind to interface address
        struct sockaddr_in6 addr6 = {};
        addr6.sin6_family = AF_INET6;
        struct ifaddrs *ifaddr, *ifa;
        if (getifaddrs(&ifaddr) == 0) {
            for (ifa = ifaddr; ifa != NULL; ifa = ifa->ifa_next) {
                if (0 != strcmp(ifa->ifa_name, if_name)) {
                    continue;
                }
                if (ifa->ifa_addr->sa_family == AF_INET6) {
                    struct sockaddr_in6 *sin6 = (struct sockaddr_in6 *)ifa->ifa_addr;
                    // Skip link-local addresses
                    if (sin6->sin6_addr.s6_addr[0] == 0xfe && sin6->sin6_addr.s6_addr[1] == 0x80) {
                        continue;
                    }
                    memcpy(&addr6, ifa->ifa_addr, ifa->ifa_addr->sa_len);
                    break;
                }
            }
            freeifaddrs(ifaddr);
        }
        
        if (addr6.sin6_family == AF_INET6) {
            if (bind(fd, (struct sockaddr *)&addr6, sizeof(addr6)) != 0) {
                errlog(g_logger, "Bind to {} failed: {}", if_name, strerror(errno));
                return -1;
            }
        }
    }
    
    return 0; // Success
}

// Set outbound interface for iOS
// Call this from Swift/Obj-C before starting the VPN
extern "C" DOBBY_EXPORT void dobby_vpn_ios_set_outbound_interface(uint32_t interface_index) {
    g_outbound_interface = interface_index;
    char if_name[IF_NAMESIZE] = "not set";
    if_indextoname(interface_index, if_name);
    infolog(g_logger, "Set outbound interface to {} ({})", interface_index, if_name);
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
        
        // 3. Parse the TOML *inside* the background thread
        toml::parse_result parsed_config = toml::parse(config_str);
        if (!parsed_config) {
            errlog(g_logger, "Failed to parse TOML config");
            return;
        }

        auto trusttunnel_config = ag::TrustTunnelConfig::build_config(parsed_config);
        ag::vpn_post_quantum_group_set_enabled(trusttunnel_config->post_quantum_group_enabled);

        ag::VpnCallbacks callbacks;
        
        // iOS-specific socket protection handler
        callbacks.protect_handler = [](ag::SocketProtectEvent *event) {
            // If Go provides a callback, use it. Otherwise, use iOS interface binding
            if (g_protect_callback) {
                event->result = g_protect_callback(event->fd);
            } else {
                event->result = vpn_ios_protect_socket(event->fd, event->peer);
            }
        };
        
        // iOS-specific certificate verification
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
        m_vpn = new dobby_vpn_context();
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
