#!/usr/bin/env bash
# Build and verify one Apple static bridge architecture with an isolated Conan cache.
set -euo pipefail

readonly requested_platform="${1:-}"
readonly go_version=1.26.8
readonly root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
case "$requested_platform" in
  ios)
    readonly platform=ios
    readonly architecture=arm64
    readonly go_arch=arm64
    readonly rust_target=aarch64-apple-ios
    readonly deployment_target=15.6
    readonly sdk=iphoneos
    readonly system_name=iOS
    readonly build_name=ios-arm64
    readonly output_directory="$root/lib/ios"
    readonly output="$output_directory/libdobby_bridge.a"
    readonly conan_lockfile="$root/scripts/pins/conan/apple-ios-arm64.lock"
    ;;
  ios-simulator-arm64)
    readonly platform=ios-simulator
    readonly architecture=arm64
    readonly go_arch=arm64
    readonly rust_target=aarch64-apple-ios-sim
    readonly deployment_target=15.6
    readonly sdk=iphonesimulator
    readonly system_name=iOS
    readonly build_name=ios-simulator-arm64
    readonly output_directory="$root/lib/ios-simulator/arm64"
    readonly output="$output_directory/libdobby_bridge.a"
    readonly conan_lockfile="$root/scripts/pins/conan/apple-ios-arm64.lock"
    ;;
  ios-simulator-amd64)
    readonly platform=ios-simulator
    readonly architecture=x86_64
    readonly go_arch=amd64
    readonly rust_target=x86_64-apple-ios
    readonly deployment_target=15.6
    readonly sdk=iphonesimulator
    readonly system_name=iOS
    readonly build_name=ios-simulator-amd64
    readonly output_directory="$root/lib/ios-simulator/amd64"
    readonly output="$output_directory/libdobby_bridge.a"
    readonly conan_lockfile="$root/scripts/pins/conan/apple-ios-arm64.lock"
    ;;
  macos)
    readonly platform=macos
    readonly architecture=arm64
    readonly go_arch=arm64
    readonly rust_target=aarch64-apple-darwin
    readonly deployment_target=12.0
    readonly sdk=macosx
    readonly system_name=Darwin
    readonly build_name=macos-arm64
    readonly output_directory="$root/lib/macos"
    readonly output="$output_directory/libdobby_bridge.a"
    readonly conan_lockfile="$root/scripts/pins/conan/apple-macos-arm64.lock"
    ;;
  *)
    echo "usage: $0 ios|ios-simulator-arm64|ios-simulator-amd64|macos" >&2
    exit 2
    ;;
esac

readonly trusttunnel="$root/TrustTunnelClient"
readonly build="$trusttunnel/build-$build_name"
readonly ios_compiler_builtins_sha256=907dea761e9fd300f3c713602b42934e33c0b480861b8cdb8b529b82a4f48402
readonly trusttunnel_cargo_lock_sha256=5dfa92024c6ff9dd09f0110fe7f094c5d2e25131787b3cdbacdafb94554b2f93
readonly conan_version=2.12.2
readonly rust_release=1.85.0
readonly rust_commit=4d91de4e48198da2e33413efdcd9cd2cc0c46688
readonly effective_cargo_home="${CARGO_HOME:-${HOME:?HOME is required}/.cargo}"
readonly effective_rustup_home="${RUSTUP_HOME:-${HOME:?HOME is required}/.rustup}"

[[ -n "${CONAN_HOME:-}" && "$CONAN_HOME" = /* ]] || {
  echo "CONAN_HOME must name an isolated absolute platform cache" >&2
  exit 2
}
[[ "$effective_cargo_home" = /* && "$effective_rustup_home" = /* ]] || {
  echo "Cargo and Rustup homes must be absolute" >&2
  exit 2
}
[[ "$root$CONAN_HOME$effective_cargo_home$effective_rustup_home" != *[$'\t\r\n ']* ]] || {
  echo "Apple build and toolchain paths must not contain whitespace" >&2
  exit 2
}
[[ ! -e "$build" ]] || {
  echo "refusing to reuse an existing build directory: $build" >&2
  exit 2
}
for tool in cmake ninja conan cargo rustc go xcrun libtool strip strings; do
  command -v "$tool" >/dev/null || { echo "missing build tool: $tool" >&2; exit 2; }
done
[[ "${GOTOOLCHAIN:-local}" == local ]] || {
  echo "GOTOOLCHAIN must be local for the pinned Go build" >&2
  exit 2
}
export GOTOOLCHAIN=local
[[ "$(go version | awk '{print $3}')" == "go$go_version" ]] || {
  echo "Go version differs from the pinned Apple archive input" >&2
  exit 1
}
[[ "$(conan --version)" == "Conan version $conan_version" ]] || {
  echo "Conan version differs from the pinned Apple archive input" >&2
  exit 1
}
rust_details="$(rustc --version --verbose)"
grep -Fxq "release: $rust_release" <<< "$rust_details" || {
  echo "Rust release differs from the pinned Apple archive input" >&2
  exit 1
}
grep -Fxq "commit-hash: $rust_commit" <<< "$rust_details" || {
  echo "Rust commit differs from the pinned Apple archive input" >&2
  exit 1
}
[[ -f "$trusttunnel/conan/settings_user.yml" && -f "$trusttunnel/cmake/conan_provider.cmake" ]] || {
  echo "run prepare_pinned_conan.py before the Apple build" >&2
  exit 2
}
grep -Fq 'DOBBY_CONAN_LOCKFILE is required' "$trusttunnel/cmake/conan_provider.cmake" || {
  echo "Apple build requires the locked Conan provider" >&2
  exit 2
}
[[ -f "$conan_lockfile" ]] || {
  echo "Apple Conan graph lock is missing" >&2
  exit 1
}
export DOBBY_CONAN_LOCKFILE="$conan_lockfile"
[[ "$(shasum -a 256 "$trusttunnel/trusttunnel/Cargo.lock" | awk '{print $1}')" == "$trusttunnel_cargo_lock_sha256" ]] || {
  echo "TrustTunnel Cargo lock differs from the pinned Apple input" >&2
  exit 1
}

# The pinned upstream checkout does not expose an extension hook. Apply one
# exact, idempotent build-copy edit rather than maintaining a fork of it.
if ! grep -Fxq 'add_subdirectory("../dobby_bridge" "dobby_bridge")' "$trusttunnel/CMakeLists.txt"; then
  sed -i '' '$a\
add_subdirectory("../dobby_bridge" "dobby_bridge")
' "$trusttunnel/CMakeLists.txt"
fi

export IPHONEOS_DEPLOYMENT_TARGET="${IPHONEOS_DEPLOYMENT_TARGET:-15.6}"
export MACOSX_DEPLOYMENT_TARGET="${MACOSX_DEPLOYMENT_TARGET:-12.0}"
[[ "$IPHONEOS_DEPLOYMENT_TARGET" == 15.6 && "$MACOSX_DEPLOYMENT_TARGET" == 12.0 ]] || {
  echo "Apple deployment environment differs from the supported contract" >&2
  exit 2
}
readonly sdkroot="$(xcrun --sdk "$sdk" --show-sdk-path)"
export SDKROOT="$sdkroot"
prefix_maps=(
  "-ffile-prefix-map=$root=/dobbyvpn/source"
  "-ffile-prefix-map=$CONAN_HOME=/dobbyvpn/conan"
)
rust_prefix_maps=(
  "--remap-path-prefix=$root=/dobbyvpn/source"
  "--remap-path-prefix=$CONAN_HOME=/dobbyvpn/conan"
  "--remap-path-prefix=$effective_cargo_home=/dobbyvpn/cargo"
  "--remap-path-prefix=$effective_rustup_home=/dobbyvpn/rustup"
)
printf -v prefix_flags '%s ' "${prefix_maps[@]}"
printf -v rust_prefix_flags '%s ' "${rust_prefix_maps[@]}"
export CFLAGS="${CFLAGS:+$CFLAGS }${prefix_flags% }"
export CXXFLAGS="${CXXFLAGS:+$CXXFLAGS }${prefix_flags% }"
export RUSTFLAGS="${RUSTFLAGS:+$RUSTFLAGS }${rust_prefix_flags% }"

configure=(
  cmake -S "$trusttunnel" -B "$build" -GNinja
  -DCMAKE_BUILD_TYPE=Release
  "-DCMAKE_OSX_ARCHITECTURES=$architecture"
  -DCMAKE_OSX_DEPLOYMENT_TARGET="$deployment_target"
  "-DCMAKE_OSX_SYSROOT=$sdkroot"
  -DCMAKE_C_COMPILER="$(xcrun --sdk "$sdk" --find clang)"
  -DCMAKE_CXX_COMPILER="$(xcrun --sdk "$sdk" --find clang++)"
  "-DCMAKE_C_FLAGS=${prefix_flags% }"
  "-DCMAKE_CXX_FLAGS=-stdlib=libc++ ${prefix_flags% }"
  -DIPV6_UNAVAILABLE=ON
  -DDOBBY_BRIDGE_STATIC=ON
  -DCARGO_EXTRA_ARGS=--locked
)
if [[ "$platform" == ios || "$platform" == ios-simulator ]]; then
  configure+=( -DCMAKE_SYSTEM_NAME="$system_name" )
fi
if command -v ccache >/dev/null; then
  configure+=( -DCMAKE_C_COMPILER_LAUNCHER=ccache -DCMAKE_CXX_COMPILER_LAUNCHER=ccache )
fi
"${configure[@]}"
cmake --build "$build" --target dobby_bridge

static_libraries=()
while IFS= read -r library; do static_libraries+=("$library"); done < <(
  find "$build" -type f -name '*.a' \
    -not -path '*/CMakeFiles/*' \
    -not -name '*.dll.a' \
    -not -name 'libdobby_bridge-merged.a' \
    -print | sort
)

conan_archive_matches_target_platform() {
  python3 - "$root/scripts" "$1" "$platform" "$architecture" <<'PY'
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, sys.argv[1])
from verify_apple_archive import PLATFORMS, parse_otool, platform_mismatches

archive = Path(sys.argv[2])
platform = sys.argv[3]
try:
    output = subprocess.run(
        ["xcrun", "otool", "-l", str(archive)],
        check=True,
        stdin=subprocess.DEVNULL,
        stdout=subprocess.PIPE,
        text=True,
    ).stdout
    records = parse_otool(output, sys.argv[4])
except (OSError, ValueError, subprocess.CalledProcessError) as error:
    print(f"error: cannot inspect Conan archive platform for {archive}: {error}", file=sys.stderr)
    raise SystemExit(2)

mismatches = platform_mismatches(records, platform)
if mismatches:
    first = mismatches[0]
    print(
        f"Skipping Conan archive for another Apple platform: {archive} "
        f"member={first.member} platform={first.platform} expected={PLATFORMS[platform]}",
        file=sys.stderr,
    )
    raise SystemExit(1)
PY
}

while IFS= read -r library; do
  if ! xcrun lipo -info "$library" 2>/dev/null | grep -q "$architecture"; then
    continue
  fi
  if conan_archive_matches_target_platform "$library"; then
    static_libraries+=("$library")
  else
    platform_check_status=$?
    if [[ "$platform_check_status" -ne 1 ]]; then
      exit "$platform_check_status"
    fi
  fi
done < <(find "$CONAN_HOME/p" -type f -name '*.a' -print | sort)
(( ${#static_libraries[@]} > 0 )) || { echo "no $architecture static libraries found" >&2; exit 1; }

# Recursive Conan cache discovery can encounter byte-identical build and
# package copies. Merging each copy only duplicates object members, so retain
# the first occurrence of every exact archive digest.
readonly digest_index="$build/static-library-sha256.txt"
: > "$digest_index"
unique_static_libraries=()
for library in "${static_libraries[@]}"; do
  digest="$(shasum -a 256 "$library" | awk '{print $1}')"
  if grep -Fqx "$digest" "$digest_index"; then
    continue
  fi
  printf '%s\n' "$digest" >> "$digest_index"
  unique_static_libraries+=( "$library" )
done
(( ${#unique_static_libraries[@]} > 0 )) || { echo "no unique static libraries found" >&2; exit 1; }

merge_libraries=( "${unique_static_libraries[@]}" )
if [[ "$platform" == ios || "$platform" == ios-simulator ]]; then
  readonly rust_target_libdir="$(rustc --print target-libdir --target "$rust_target")"
  compiler_builtins=( "$rust_target_libdir"/libcompiler_builtins-*.rlib )
  (( ${#compiler_builtins[@]} == 1 )) && [[ -f "${compiler_builtins[0]}" ]] || {
    echo "expected exactly one pinned iOS compiler-builtins rlib" >&2
    exit 1
  }

  readonly sanitized_directory="$build/sanitized-static-inputs"
  mkdir "$sanitized_directory"
  expected_compiler_builtins_sha256="$ios_compiler_builtins_sha256"
  if [[ "$platform" == ios-simulator ]]; then
    # Simulator targets use different immutable rlibs from the physical iOS
    # target. The exact Rust release and commit are verified above.
    expected_compiler_builtins_sha256="$(shasum -a 256 "${compiler_builtins[0]}" | awk '{print $1}')"
  fi
  merge_libraries=()
  input_number=0
  for library in "${unique_static_libraries[@]}"; do
    input_number=$((input_number + 1))
    original_digest="$(shasum -a 256 "$library" | awk '{print $1}')"
    staged="$sanitized_directory/input-$input_number.a"
    echo "Checking Apple static archive input: $library"
    cp -p "$library" "$staged"
    python3 "$root/scripts/prune_apple_compiler_builtins.py" \
      --archive "$staged" \
      --compiler-builtins "${compiler_builtins[0]}" \
      --expected-compiler-builtins-sha256 "$expected_compiler_builtins_sha256" \
      --platform "$platform" \
      --architecture "$architecture" \
      --maximum-deployment-target "$deployment_target"
    [[ "$(shasum -a 256 "$library" | awk '{print $1}')" == "$original_digest" ]] || {
      echo "Apple input sanitizer modified a Conan/build-cache archive" >&2
      exit 1
    }
    merge_libraries+=( "$staged" )
  done
fi

mkdir -p "$output_directory"
readonly merged="$build/libdobby_bridge-merged.a"
libtool -static -D -o "$merged" "${merge_libraries[@]}"
strip -S -D "$merged"
install -m 0644 "$merged" "$output"
python3 "$root/scripts/verify_apple_archive.py" \
  --archive "$output" \
  --platform "$platform" \
  --architecture "$architecture" \
  --maximum-deployment-target "$deployment_target" \
  --canonicalize-metadata
readonly strings_inventory="$build/archive-strings.txt"
strings "$output" > "$strings_inventory" || {
  echo "Apple archive local-path scan failed" >&2
  exit 1
}
if grep -F \
  -e "$root" -e "$CONAN_HOME" -e "$effective_cargo_home" -e "$effective_rustup_home" \
  -e '/Users/' -e '/home/' \
  "$strings_inventory" >/dev/null; then
  echo "Apple archive contains an unremapped local build path" >&2
  exit 1
fi

if [[ "$platform" == ios-simulator ]]; then
  readonly simulator_bridge_link="$root/lib/ios-simulator/libdobby_bridge.a"
  readonly simulator_bridge_backup="$build/libdobby_bridge-before-consumer.a"
  simulator_bridge_link_created=0
  simulator_bridge_backup_created=0
  restore_simulator_consumer_bridge() {
    local status=$?
    trap - EXIT
    if [[ "$simulator_bridge_link_created" == 1 ]]; then
      rm -f "$simulator_bridge_link" || status=$?
    fi
    if [[ "$simulator_bridge_backup_created" == 1 ]]; then
      mv "$simulator_bridge_backup" "$simulator_bridge_link" || status=$?
    fi
    exit "$status"
  }
  trap restore_simulator_consumer_bridge EXIT
  if [[ -e "$simulator_bridge_link" || -L "$simulator_bridge_link" ]]; then
    mv "$simulator_bridge_link" "$simulator_bridge_backup"
    simulator_bridge_backup_created=1
  fi
  ln -s "$output" "$simulator_bridge_link"
  simulator_bridge_link_created=1
fi

if [[ "$platform" == ios || "$platform" == ios-simulator ]]; then
  (
    cd "$root/examples"
    minimum_flag=-miphoneos-version-min
    if [[ "$platform" == ios-simulator ]]; then minimum_flag=-mios-simulator-version-min; fi
    simulator_tags=static
    if [[ "$platform" == ios-simulator ]]; then simulator_tags='static,simulator'; fi
    CGO_ENABLED=1 GOOS=ios GOARCH="$go_arch" \
      CC="$(xcrun --sdk "$sdk" --find clang) -arch $architecture -isysroot $(xcrun --sdk "$sdk" --show-sdk-path) $minimum_flag=$deployment_target" \
      CXX="$(xcrun --sdk "$sdk" --find clang++) -arch $architecture -isysroot $(xcrun --sdk "$sdk" --show-sdk-path) $minimum_flag=$deployment_target" \
      go build -trimpath -tags "$simulator_tags" -o "$build/example-ios"
  )
else
  (
    cd "$root/examples"
    CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 \
      CC="$(xcrun --sdk macosx --find clang) -arch arm64 -isysroot $(xcrun --sdk macosx --show-sdk-path) -mmacosx-version-min=$deployment_target" \
      CXX="$(xcrun --sdk macosx --find clang++) -arch arm64 -isysroot $(xcrun --sdk macosx --show-sdk-path) -mmacosx-version-min=$deployment_target" \
      go build -trimpath -tags static -o "$build/example-macos-arm64"
  )
fi

shasum -a 256 "$output"
