# dobby_bridge/CMakeLists.txt
cmake_minimum_required(VERSION 3.16)
project(DobbyBridge)

# 1. Bring in the unmodified TrustTunnel build system
add_subdirectory(../vendor/TrustTunnelClient/platform/windows trusttunnel_build)

# 2. Define your custom shared library
add_library(dobby_bridge SHARED dobby_bridge.cpp)

# 3. Link AdGuard's core static libraries to your bridge
target_link_libraries(dobby_bridge PRIVATE
    vpnlibs_trusttunnel
    vpnlibs_core
    vpnlibs_net
    vpnlibs_common
)
