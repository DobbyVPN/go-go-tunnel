# We are injected into AdGuard's tree, so we have direct access to vpn_easy_a!
add_library(dobby_bridge SHARED dobby_bridge.cpp)
target_link_libraries(dobby_bridge PRIVATE vpn_easy_a)
set_target_properties(dobby_bridge PROPERTIES OUTPUT_NAME "dobby_bridge")