#!/bin/bash
cp README.md ./npm
cp LICENSE ./npm
(cd npm && npm publish)