#!/bin/sh

for f in bin/pprof-*; do shasum -a 256 $f > $f.sha256; done