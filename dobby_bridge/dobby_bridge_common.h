#pragma once

#include <stddef.h>
#include <stdint.h>

#if defined(_WIN32)
    #define DOBBY_EXPORT __declspec(dllexport)
#else
    #define DOBBY_EXPORT __attribute__((visibility("default")))
#endif

#ifdef __cplusplus
extern "C" {
#endif

// Callback types
typedef void (*dobby_on_state_changed_t)(void *arg, int new_state_description);
typedef void (*dobby_on_log_message_t)(int level, const char *message);

// Socket protection callback - user can provide their own implementation
// Returns 0 on success, non-zero on failure
typedef int (*dobby_on_protect_socket_t)(int fd);

// Core API functions
DOBBY_EXPORT void dobby_vpn_set_log_callback(dobby_on_log_message_t cb);
DOBBY_EXPORT void dobby_vpn_set_protect_callback(dobby_on_protect_socket_t cb);

DOBBY_EXPORT void dobby_vpn_start(
        const char *toml_config, 
        dobby_on_state_changed_t state_changed_cb, 
        void *state_changed_cb_arg);

DOBBY_EXPORT void dobby_vpn_stop();

#ifdef __cplusplus
}
#endif
