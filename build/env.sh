#!/bin/sh

set -e

if [ ! -f "build/env.sh" ]; then
    echo "$0 must be run from the root of the repository."
    exit 2
fi

# Create fake Go workspace if it doesn't exist yet.
workspace="$PWD/build/_workspace"
root="$PWD"
gttcdir="$workspace/src/github.com/TTCECO"
if [ ! -L "$gttcdir/gttc" ]; then
    mkdir -p "$gttcdir"
    cd "$gttcdir"
    ln -s ../../../../../. gttc
    cd "$root"
fi

# Set up the environment to use the workspace.
GOPATH="$workspace"
export GOPATH

# Run the command inside the workspace.
cd "$gttcdir/gttc"
PWD="$gttcdir/gttc"

# Launch the arguments with the configured environment.
exec "$@"
