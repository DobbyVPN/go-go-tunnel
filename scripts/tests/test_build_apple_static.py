from pathlib import Path
import unittest


SCRIPT = Path(__file__).resolve().parents[1] / "build_apple_static.sh"


class AppleStaticBuildContractTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls) -> None:
        cls.source = SCRIPT.read_text(encoding="utf-8")

    def test_both_consumer_links_use_an_explicit_sdk_sysroot(self) -> None:
        self.assertGreaterEqual(self.source.count("-isysroot $(xcrun --sdk"), 2)
        self.assertIn('sdk=iphonesimulator', self.source)

    def test_native_build_exports_selected_sdk_to_rust_and_conan(self) -> None:
        self.assertIn('readonly sdkroot="$(xcrun --sdk "$sdk" --show-sdk-path)"', self.source)
        self.assertIn('export SDKROOT="$sdkroot"', self.source)
        self.assertIn('"-DCMAKE_OSX_SYSROOT=$sdkroot"', self.source)

    def test_native_compilers_remap_source_and_dependency_paths(self) -> None:
        for required in (
            "-ffile-prefix-map=$root=/dobbyvpn/source",
            "-ffile-prefix-map=$CONAN_HOME=/dobbyvpn/conan",
            "--remap-path-prefix=$root=/dobbyvpn/source",
            "--remap-path-prefix=$CONAN_HOME=/dobbyvpn/conan",
            '${CARGO_HOME:-${HOME:?HOME is required}/.cargo}',
            '${RUSTUP_HOME:-${HOME:?HOME is required}/.rustup}',
            "--remap-path-prefix=$effective_cargo_home=/dobbyvpn/cargo",
            "--remap-path-prefix=$effective_rustup_home=/dobbyvpn/rustup",
            "-e '/Users/' -e '/home/'",
            "Apple archive contains an unremapped local build path",
        ):
            self.assertIn(required, self.source)

    def test_exact_rust_toolchain_is_checked_for_both_platforms(self) -> None:
        platform_branch = self.source.index('if [[ "$platform" == ios || "$platform" == ios-simulator ]]')
        for required in (
            "rustc --version --verbose",
            'release: $rust_release',
            'commit-hash: $rust_commit',
        ):
            self.assertLess(self.source.index(required), platform_branch)

    def test_exact_conan_version_is_checked_for_both_platforms(self) -> None:
        platform_branch = self.source.index('if [[ "$platform" == ios || "$platform" == ios-simulator ]]')
        for required in (
            "readonly conan_version=2.12.2",
            '$(conan --version)',
            'Conan version $conan_version',
        ):
            self.assertLess(self.source.index(required), platform_branch)

    def test_trusttunnel_rust_uses_immutable_lock(self) -> None:
        self.assertIn(
            "5dfa92024c6ff9dd09f0110fe7f094c5d2e25131787b3cdbacdafb94554b2f93",
            self.source,
        )
        self.assertIn("-DCARGO_EXTRA_ARGS=--locked", self.source)

    def test_conan_graph_uses_platform_lock(self) -> None:
        self.assertIn('apple-ios-arm64.lock', self.source)
        self.assertIn('apple-macos-arm64.lock', self.source)
        self.assertEqual(
            self.source.count('readonly conan_lockfile="$root/scripts/pins/conan/apple-macos-arm64.lock"'),
            2,
        )
        self.assertIn('export DOBBY_CONAN_LOCKFILE="$conan_lockfile"', self.source)
        self.assertIn('DOBBY_CONAN_LOCKFILE is required', self.source)
        self.assertIn('Apple build requires the locked Conan provider', self.source)

    def test_macos_builds_are_separate_thin_archives_with_matching_consumers(self) -> None:
        for required in (
            'macos-arm64)',
            'macos-x86_64)',
            'readonly architecture=x86_64',
            'readonly go_arch=amd64',
            'readonly rust_target=x86_64-apple-darwin',
            'readonly output_directory="$root/lib/macos/x86_64"',
            "grep -Fxq 'arch=x86_64' \"$CONAN_HOME/profiles/default\"",
            'native Intel macOS runner',
            'GOOS=darwin GOARCH="$go_arch"',
            '-arch $architecture',
        ):
            self.assertIn(required, self.source)
        self.assertNotIn('conan profile update', self.source)

    def test_conan_archives_are_filtered_by_apple_platform(self) -> None:
        self.assertIn('conan_archive_matches_target_platform "$library"', self.source)
        self.assertIn('parse_otool(output, sys.argv[4])', self.source)
        self.assertIn('Skipping Conan archive for another Apple platform:', self.source)
        self.assertIn('platform_mismatches(records, platform)', self.source)
        self.assertGreaterEqual(self.source.count('--architecture "$architecture"'), 2)

    def test_simulator_build_uses_simulator_sdk_and_build_tags(self) -> None:
        for required in (
            "ios-simulator-arm64)",
            "ios-simulator-amd64)",
            "sdk=iphonesimulator",
            "-mios-simulator-version-min",
            "simulator_tags='static,simulator'",
        ):
            self.assertIn(required, self.source)

    def test_consumer_links_stage_the_current_archive_and_restore_existing(self) -> None:
        for required in (
            'if [[ "$platform" == ios-simulator || ( "$platform" == macos && "$architecture" == x86_64 ) ]]; then',
            'consumer_bridge_link="$root/lib/ios-simulator/libdobby_bridge.a"',
            'consumer_bridge_link="$root/lib/macos/libdobby_bridge.a"',
            'consumer_bridge_backup="$build/libdobby_bridge-before-consumer.a"',
            'if [[ -e "$consumer_bridge_link" || -L "$consumer_bridge_link" ]]; then',
            'mv "$consumer_bridge_link" "$consumer_bridge_backup"',
            'ln -s "$output" "$consumer_bridge_link"',
            'restore_consumer_bridge() ',
            'rm -f "$consumer_bridge_link"',
            'mv "$consumer_bridge_backup" "$consumer_bridge_link"',
        ):
            self.assertIn(required, self.source)
        staging = self.source.index('ln -s "$output" "$consumer_bridge_link"')
        ios_consumer = self.source.index('CGO_ENABLED=1 GOOS=ios GOARCH="$go_arch"')
        macos_consumer = self.source.index('CGO_ENABLED=1 GOOS=darwin GOARCH="$go_arch"')
        self.assertLess(staging, ios_consumer)
        self.assertLess(staging, macos_consumer)
        self.assertLess(
            self.source.index('trap restore_consumer_bridge EXIT'),
            self.source.index('mv "$consumer_bridge_link" "$consumer_bridge_backup"'),
        )
        self.assertLess(
            self.source.index('rm -f "$consumer_bridge_link"'),
            self.source.index('mv "$consumer_bridge_backup" "$consumer_bridge_link"'),
        )

    def test_final_archive_verification_precedes_consumer_link(self) -> None:
        verifier = self.source.index("scripts/verify_apple_archive.py")
        privacy_scan = self.source.index("Apple archive contains an unremapped local build path")
        consumer = self.source.index("CGO_ENABLED=1 GOOS=ios")
        self.assertLess(verifier, privacy_scan)
        self.assertLess(privacy_scan, consumer)

    def test_archive_merge_selects_only_the_requested_architecture(self) -> None:
        self.assertIn(
            'libtool -static -D -arch_only "$architecture" -o "$merged"',
            self.source,
        )

    def test_final_archive_uses_deterministic_apple_tools(self) -> None:
        self.assertIn('strip -S -D "$merged"', self.source)
        self.assertIn("--canonicalize-metadata", self.source)
        self.assertNotIn('libtool -static -o "$merged"', self.source)
        self.assertIn('scripts/verify_apple_archive.py', self.source)


if __name__ == "__main__":
    unittest.main()
