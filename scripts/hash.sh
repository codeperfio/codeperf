#!/bin/sh

for f in bin/codeperf-*; do shasum -a 256 $f > $f.sha256; done