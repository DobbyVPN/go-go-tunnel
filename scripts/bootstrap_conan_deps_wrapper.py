#!/usr/bin/env python3

"""
Wrapper script for bootstrap_conan_deps.py that patches the script
to support checking out a specific commit before reading conandata.yml.

This is necessary because the current master branch of NativeLibsCommon
has deleted conandata.yml, and we need to use a specific commit where
it still exists.
"""

import os
import subprocess
import sys
import tempfile
import shutil

# Get the directory where this script is located
script_dir = os.path.dirname(os.path.realpath(__file__))
# TrustTunnelClient is at the root level, not inside scripts/
project_root = os.path.dirname(script_dir)
bootstrap_script = os.path.join(project_root, "TrustTunnelClient", "scripts", "bootstrap_conan_deps.py")
trusttunnel_scripts_dir = os.path.join(project_root, "TrustTunnelClient", "scripts")

def patch_bootstrap_script(nlc_commit):
    """
    Patch the bootstrap script to checkout a specific commit after cloning NLC.
    Returns the path to the patched script.
    """
    # Read the original script
    with open(bootstrap_script, 'r') as f:
        content = f.read()
    
    # Check if already patched with this commit
    if f'checkout {nlc_commit}' in content:
        # Already patched, return original
        return bootstrap_script
    
    # Patch work_dir and project_dir to use the correct paths
    # The patched script runs from a temp directory, so we need to fix the paths
    old_work_dir = 'work_dir = os.path.dirname(os.path.realpath(__file__))'
    new_work_dir = f'work_dir = r"{trusttunnel_scripts_dir}"'
    content = content.replace(old_work_dir, new_work_dir)
    
    # Add checkout logic after git clone
    # Find the line that does the git clone and add checkout after it
    old_clone_line = '    subprocess.run(["git", "clone", nlc_url, nlc_dir], check=True)\n    os.chdir(nlc_dir)'
    new_clone_line = f'    subprocess.run(["git", "clone", nlc_url, nlc_dir], check=True)\n    os.chdir(nlc_dir)\n    \n    # Checkout specific commit to access conandata.yml\n    subprocess.run(["git", "checkout", "{nlc_commit}"], check=True)'
    
    content = content.replace(old_clone_line, new_clone_line)
    
    # Write to a temporary file
    temp_dir = tempfile.mkdtemp()
    patched_script = os.path.join(temp_dir, "bootstrap_conan_deps_patched.py")
    with open(patched_script, 'w') as f:
        f.write(content)
    
    return patched_script

def main():
    # Parse arguments
    nlc_url = sys.argv[1] if len(sys.argv) > 1 else 'https://github.com/AdguardTeam/NativeLibsCommon.git'
    dns_libs_url = sys.argv[2] if len(sys.argv) > 2 else 'https://github.com/AdguardTeam/DnsLibs.git'
    nlc_commit = sys.argv[3] if len(sys.argv) > 3 else None
    
    # Change to the TrustTunnelClient/scripts directory
    original_dir = os.getcwd()
    os.chdir(trusttunnel_scripts_dir)
    
    try:
        if nlc_commit:
            # Patch the bootstrap script to checkout the specific commit
            patched_script = patch_bootstrap_script(nlc_commit)
            
            try:
                # Run the patched script
                result = subprocess.run([sys.executable, patched_script, nlc_url, dns_libs_url], check=True)
                sys.exit(result.returncode)
            finally:
                # Clean up temporary file
                if patched_script != bootstrap_script:
                    temp_dir = os.path.dirname(patched_script)
                    shutil.rmtree(temp_dir, ignore_errors=True)
        else:
            # No commit specified, just call the bootstrap script normally
            result = subprocess.run([sys.executable, bootstrap_script, nlc_url, dns_libs_url], check=True)
            sys.exit(result.returncode)
    finally:
        # Restore original directory
        os.chdir(original_dir)

if __name__ == "__main__":
    main()
