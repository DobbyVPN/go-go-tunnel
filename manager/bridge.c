#include "_cgo_export.h"
#include <stdlib.h>

// C "Gateway" functions that bridge C callbacks to Go
void c_state_changed(void* arg, int state) {
    go_state_changed(arg, state);
}

void c_log_message(int level, const char* msg) {
    go_log_message(level, (char*)msg);
}

int c_protect_cb(int fd) {
    return go_protect_socket(fd);
}
