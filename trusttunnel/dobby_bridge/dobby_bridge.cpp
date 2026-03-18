// 1. Include their unmodified internal headers
#include "../../vendor/TrustTunnelClient/common/include/common/logger.h"
#include "../../vendor/TrustTunnelClient/vpn/trusttunnel/include/vpn/trusttunnel/client.h"
#include "../../vendor/TrustTunnelClient/vpn/trusttunnel/include/vpn/trusttunnel/config.h"

// 2. Include your C-API header for Go
#include "dobby_bridge.h"

// 3. Implement your custom logic
static on_log_message_t g_log_callback = nullptr;

void dobby_set_log_callback(on_log_message_t cb) {
    g_log_callback = cb;
    ag::Logger::set_callback([](ag::LogLevel log_level, std::string_view message) {
        if (g_log_callback) {
            std::string msg_str(message);
            g_log_callback(static_cast<int>(log_level), msg_str.c_str());
        }
    });
}

// ... implement dobby_start_vpn and dobby_set_protect_callback here ...