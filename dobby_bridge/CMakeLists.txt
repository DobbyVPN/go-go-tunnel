# dobby_bridge: Go-compatible C wrapper around TrustTunnelClient
#
# We are injected into TrustTunnelClient's CMake tree via:
#   add_subdirectory("../dobby_bridge" "dobby_bridge")
# appended to TrustTunnelClient/CMakeLists.txt
#
# The vpnlibs_* targets and all their Conan dependencies (dns-libs, openssl,
# libevent, ldns, etc.) are STATIC libraries. When we build dobby_bridge as
# a SHARED library, we MUST force the linker to include ALL object files from
# every static dependency. Otherwise the linker strips "unused" symbols like
# global C++ static initializers (DNS provider factories, protocol registries)
# and the library crashes at runtime.

cmake_minimum_required(VERSION 3.24)

# Option to build static or shared library
option(BUILD_STATIC_LIBS "Build static library instead of shared" OFF)

# Disable security cookies for MSVC to maintain compatibility with Cgo/MinGW
if(WIN32 AND MSVC)
    add_compile_options(/GS-)
endif()

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

# Create library (static or shared based on option)
if(BUILD_STATIC_LIBS)
    add_library(dobby_bridge STATIC ${DOBBY_BRIDGE_SOURCE})
    message(STATUS "Building dobby_bridge as STATIC library")
else()
    add_library(dobby_bridge SHARED ${DOBBY_BRIDGE_SOURCE})
    message(STATUS "Building dobby_bridge as SHARED library")
endif()

# Include the common header
target_include_directories(dobby_bridge PUBLIC ${CMAKE_CURRENT_SOURCE_DIR})

# ---------------------------------------------------------------------------
# Linking Strategy
# ---------------------------------------------------------------------------
# When building SHARED: we need WHOLE_ARCHIVE on ALL static libs so that
# no symbols (especially global C++ static initializers) get stripped.
#
# The Conan packages (dns-libs, openssl, libevent, ldns, quiche, etc.) are
# resolved via find_package() and exposed as IMPORTED STATIC targets. They
# are transitively pulled in by vpnlibs_trusttunnel -> vpnlibs_core ->
# vpnlibs_net -> vpnlibs_common.
#
# We use $<LINK_LIBRARY:WHOLE_ARCHIVE,...> on the top-level vpnlibs targets.
# CMake 3.24+ will propagate WHOLE_ARCHIVE to their transitive STATIC deps.
# ---------------------------------------------------------------------------

if(NOT BUILD_STATIC_LIBS)
    # SHARED build: force all symbols in
    target_link_libraries(dobby_bridge PRIVATE
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,vpnlibs_trusttunnel>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,vpnlibs_core>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,vpnlibs_net>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,vpnlibs_common>"
    )

    # The Conan IMPORTED targets (dns-libs::dns-libs, openssl::openssl, etc.)
    # are already linked transitively by the vpnlibs_* targets above.
    # However, WHOLE_ARCHIVE alone may not pull in transitive IMPORTED targets.
    # We must explicitly link them so they end up in the final .so/.dll/.dylib.
    #
    # Re-find all required packages (they were already found by the parent CMake)
    # and link them with WHOLE_ARCHIVE too.
    find_package(dns-libs REQUIRED)
    find_package(native_libs_common REQUIRED)
    find_package(libevent REQUIRED)
    find_package(OpenSSL REQUIRED)
    find_package(ldns REQUIRED)
    find_package(nghttp2 REQUIRED)
    find_package(brotli REQUIRED)
    find_package(http_parser REQUIRED)
    find_package(klib REQUIRED)
    find_package(ZLIB REQUIRED)
    find_package(ngtcp2 REQUIRED)
    find_package(magic_enum REQUIRED)
    find_package(cxxopts REQUIRED)
    find_package(tomlplusplus REQUIRED)
    find_package(nlohmann_json REQUIRED)

    # Link all Conan static deps with WHOLE_ARCHIVE
    target_link_libraries(dobby_bridge PRIVATE
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,dns-libs::dns-libs>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,native_libs_common::native_libs_common>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,libevent::libevent>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,openssl::openssl>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,ldns::ldns>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,nghttp2::nghttp2>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,brotli::brotli>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,http_parser::http_parser>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,ngtcp2::ngtcp2>"
        "$<LINK_LIBRARY:WHOLE_ARCHIVE,ZLIB::ZLIB>"
    )

    # quiche is optional (disabled on mips)
    find_package(quiche QUIET)
    if(quiche_FOUND)
        target_link_libraries(dobby_bridge PRIVATE
            "$<LINK_LIBRARY:WHOLE_ARCHIVE,quiche::quiche>"
        )
    endif()

    # Header-only / interface libraries don't need WHOLE_ARCHIVE
    target_link_libraries(dobby_bridge PRIVATE
        klib::klib
        magic_enum::magic_enum
        cxxopts::cxxopts
        tomlplusplus::tomlplusplus
        nlohmann_json::nlohmann_json
    )
else()
    # STATIC build: normal linking, consumer will resolve symbols
    target_link_libraries(dobby_bridge PRIVATE
        vpnlibs_trusttunnel
        vpnlibs_core
        vpnlibs_net
        vpnlibs_common
    )
endif()

# Platform-specific linking
if(WIN32)
    # Windows-specific system libraries
    target_link_libraries(dobby_bridge PRIVATE
        ws2_32
        iphlpapi
        wsock32
        ntdll
        crypt32
        secur32
        userenv
        advapi32
        bcrypt
        Fwpuclnt
        Version
    )
    if(MSVC)
        target_link_libraries(dobby_bridge PRIVATE Ws2_32)
    endif()
elseif(APPLE)
    # macOS/iOS require Apple's native networking and security frameworks
    target_link_libraries(dobby_bridge PRIVATE
        "-framework CoreFoundation"
        "-framework Security"
        "-framework Network"
    )
    if(NOT CMAKE_SYSTEM_NAME STREQUAL "iOS")
        target_link_libraries(dobby_bridge PRIVATE
            "-framework Foundation"
            "-framework SystemConfiguration"
        )
    endif()
    if(IOS)
        target_link_libraries(dobby_bridge PRIVATE
            "-framework Foundation"
        )
    endif()
    # resolv is needed for DNS on non-Android Unix
    target_link_libraries(dobby_bridge PRIVATE resolv)
elseif(ANDROID)
    # Android-specific linking
    target_link_libraries(dobby_bridge PRIVATE log)
else()
    # Linux
    target_link_libraries(dobby_bridge PRIVATE resolv pthread dl)
endif()

# Set output name
set_target_properties(dobby_bridge PROPERTIES OUTPUT_NAME "dobby_bridge")

# Set C++ standard and hide visibility to prevent ABI clashes
set_target_properties(dobby_bridge PROPERTIES
    CXX_STANDARD 20
    CXX_STANDARD_REQUIRED ON
    CXX_VISIBILITY_PRESET hidden
    C_VISIBILITY_PRESET hidden
    VISIBILITY_INLINES_HIDDEN ON
)

# Force static linking of C/C++ runtimes into the shared library
if(NOT BUILD_STATIC_LIBS)
    if(WIN32 AND MSVC)
        # Force MSVC to use Static CRT (/MT) for the DLL
        set_property(TARGET dobby_bridge PROPERTY MSVC_RUNTIME_LIBRARY "MultiThreaded$<$<CONFIG:Debug>:Debug>")
    elseif(CMAKE_SYSTEM_NAME STREQUAL "Linux")
        # Statically link libc++ and libgcc into the .so
        target_link_options(dobby_bridge PRIVATE
            "-static-libstdc++"
            "-static-libgcc"
            "-Wl,--exclude-libs,ALL"
        )
    endif()
endif()

# Platform-specific compile definitions
if(WIN32)
    target_compile_definitions(dobby_bridge PRIVATE _WIN32_WINNT=0x0601)
elseif(ANDROID)
    target_compile_definitions(dobby_bridge PRIVATE ANDROID)
elseif(IOS)
    target_compile_definitions(dobby_bridge PRIVATE IOS)
endif()
