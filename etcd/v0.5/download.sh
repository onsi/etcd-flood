#!/usr/bin/env bash
if [ ! -e ./etcd ]; then
  echo "Downloading v0.5.0"
  wget https://github.com/coreos/etcd/releases/download/v0.5.0-alpha.4/etcd-v0.5.0-alpha.4-darwin-amd64.zip
  unzip etcd-v0.5.0-alpha.4-darwin-amd64.zip
  mv ./etcd-v0.5.0-alpha.4-darwin-amd64/etcd ./etcd
  rm -rf ./etcd-v0.5.0-alpha.4-darwin-amd64 etcd-v0.5.0-alpha.4-darwin-amd64.zip
fi
