# Diff
Diff takes two paths as input and checks them for differences. If both paths are files then diff reports whether the files are different. If they are directories, it reports differences in contents of the directories.

## Installation
Either download a release directly from the [releases](https://github.com/samiksome92/diff/releases) page or use Go:

    go install github.com/samiksome92/diff@latest

## Usage
    diff [flags] path1 path2

The flags are:

    -h, --help        Print this help.
    -r, --recursive   Recursively compare directories.

Diff's reporting is not provided in any specific order and may vary across runs as it parallelizes comparisons.
