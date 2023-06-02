#!/bin/bash
cp README.md ./npm
cp LICENSE ./npm
npm publish
(cd npm && npm publish)