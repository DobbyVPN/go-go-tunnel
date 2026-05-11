# We are injected into AdGuard's tree, so we have direct access to their core targets!

# Platform-specific source file selection
if(WIN32)
    set(DOBBY_BRIDGE_SOURCE dobby_bridge_windows.cpp)
elseif(ANDROID)
    set(DOBBY_BRIDGE_SOURCE dobby_bridge_android.cpp)
elseif(IOS)
    set(DOBBY_BRIDGE_SOURCE dobby_bridge_ios.mm)
else()
    # Linux and macOS (desktop)
    set(DOBBY_BRIDGE_SOURCE dobby_bridge_unix.cpp)
endif()

add_library(dobby_bridge SHARED ${DOBBY_BRIDGE_SOURCE})

# Include the common header
target_include_directories(dobby_bridge PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

# Link directly to the core engine, completely bypassing their broken vpn_easy wrapper
target_link_libraries(dobby_bridge PRIVATE
    vpnlibs_trusttunnel
    vpnlibs_core
    vpnlibs_net
    vpnlibs_common
)

# Platform-specific linking
if(WIN32)
    # Windows-specific libraries
    target_link_libraries(dobby_bridge PRIVATE
        ws2_32
        iphlpapi
    )
elseif(APPLE)
    # macOS/iOS require Apple's native networking and security frameworks
    target_link_libraries(dobby_bridge PRIVATE
        "-framework CoreFoundation"
        "-framework Security"
        "-framework Network"
    )
    if(IOS)
        # iOS-specific frameworks
        target_link_libraries(dobby_bridge PRIVATE
            "-framework Foundation"
        )
    endif()
elseif(ANDROID)
    # Android-specific linking
    target_link_libraries(dobby_bridge PRIVATE
        log
    )
endif()

# Set output name
set_target_properties(dobby_bridge PROPERTIES OUTPUT_NAME "dobby_bridge")

# Set C++ standard
set_target_properties(dobby_bridge PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
)

# Platform-specific compile definitions
if(WIN32)
    target_compile_definitions(dobby_bridge PRIVATE _WIN32_WINNT=0x0601)
elseif(ANDROID)
    target_compile_definitions(dobby_bridge PRIVATE ANDROID)
elseif(IOS)
    target_compile_definitions(dobby_bridge PRIVATE IOS)
endif()
