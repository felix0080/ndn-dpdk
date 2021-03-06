#!/bin/bash
set -e
set -o pipefail
cd "$( dirname "${BASH_SOURCE[0]}" )"/..

(
  cd csrc
  echo '# <auto-generated> mk/update-list.sh'
  echo 'csrc = files('
  find -name '*.c' -printf '%P\n' | sed "s|.*|'\0'|" | paste -sd,
  echo ')'
) > csrc/meson.build

(
  echo '# <auto-generated> mk/update-list.sh'
  echo 'cgoflags_dirs = ['
  git grep -l '^import "C"$' '**/*.go' | xargs dirname | sort -u | sed "s|.*|'\0'|" | paste -sd,
  echo ']'
  echo 'cgostruct_dirs = ['
  git ls-files '**/cgostruct.in.go' | xargs dirname | sed "s|.*|'\0'|" | paste -sd,
  echo ']'
) > mk/meson.build
