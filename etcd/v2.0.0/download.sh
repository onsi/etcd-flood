#!/usr/bin/env bash
if [ ! -e ./etcd ]; then
  echo "Downloading v2.0.0"
  wget https://github.com/coreos/etcd/releases/download/v2.0.0/etcd-v2.0.0-darwin-amd64.zip
  unzip etcd-v2.0.0-darwin-amd64.zip
  mv ./etcd-v2.0.0-darwin-amd64/etcd ./etcd
  rm -rf ./etcd-v2.0.0-darwin-amd64 etcd-v2.0.0-darwin-amd64.zip
fi
