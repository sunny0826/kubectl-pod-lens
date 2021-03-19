#!/bin/bash

set -e

echo "git clone"
git config --global user.email "actions@github.com"
git config --global user.name "gh-actions"
git clone --single-branch --depth 1 https://github.com/sunny0826/pod-lens.github.io.git

echo "clear en docs"
rm -rf pod-lens.github.io/docs/*
echo "clear zh docs"
rm -rf pod-lens.github.io/i18n/zh/docusaurus-plugin-content-docs/current/*

echo "update docs"
cp -R doc/en/* pod-lens.github.io/docs/
cp -R doc/zh/* pod-lens.github.io/i18n/zh/docusaurus-plugin-content-docs/current/

echo "git push"
cd pod-lens.github.io
git add .
git commit -m "github action auto sync"
git push origin master