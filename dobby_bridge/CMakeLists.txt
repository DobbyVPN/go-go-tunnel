cmake_minimum_required(VERSION 3.24)
project(dobby_bridge C CXX)

set(CMAKE_CXX_STANDARD 20)
set(CMAKE_CXX_STANDARD_REQUIRED ON)
set(CMAKE_POSITION_INDEPENDENT_CODE ON)

option(DOBBY_BRIDGE_STATIC "Build dobby_bridge as a static library" OFF)

# Select platform-specific source file
if(WIN32)
    set(DOBBY_BRIDGE_SOURCES dobby_bridge_windows.cpp)
elseif(ANDROID)
    set(DOBBY_BRIDGE_SOURCES dobby_bridge_android.cpp)
elseif(CMAKE_SYSTEM_NAME STREQUAL "iOS")
    set(DOBBY_BRIDGE_SOURCES dobby_bridge_ios.mm)
else()
    # Linux, macOS, and other Unix-like systems
    set(DOBBY_BRIDGE_SOURCES dobby_bridge_unix.cpp)
endif()

# Build library type
if(DOBBY_BRIDGE_STATIC)
    add_library(dobby_bridge STATIC ${DOBBY_BRIDGE_SOURCES})
else()
    add_library(dobby_bridge SHARED ${DOBBY_BRIDGE_SOURCES})
endif()

target_include_directories(dobby_bridge PRIVATE ${CMAKE_CURRENT_SOURCE_DIR})

# Link TrustTunnel core — brings in all transitive deps (conan packages, etc.)
target_link_libraries(dobby_bridge PRIVATE vpnlibs_trusttunnel)

# Platform-specific system libraries
if(WIN32)
    target_link_libraries(dobby_bridge PRIVATE
        ws2_32 crypt32 userenv version Fwpuclnt Advapi32
    )
    if(MSVC)
        set_target_properties(dobby_bridge PROPERTIES
            MSVC_RUNTIME_LIBRARY "MultiThreaded"
        )
    endif()
elseif(APPLE)
    target_link_libraries(dobby_bridge PRIVATE
        "-framework Foundation"
        "-framework Security"
        "-framework CoreFoundation"
        "-framework SystemConfiguration"
    )
    if(CMAKE_SYSTEM_NAME STREQUAL "iOS")
        target_link_libraries(dobby_bridge PRIVATE "-framework NetworkExtension")
    endif()
elseif(ANDROID)
    target_link_libraries(dobby_bridge PRIVATE log android)
else()
    # Linux
    target_link_libraries(dobby_bridge PRIVATE pthread dl)
endif()

# Symbol visibility — only DOBBY_EXPORT functions are visible
set_target_properties(dobby_bridge PROPERTIES
    C_VISIBILITY_PRESET hidden
    CXX_VISIBILITY_PRESET hidden
    VISIBILITY_INLINES_HIDDEN ON
)
