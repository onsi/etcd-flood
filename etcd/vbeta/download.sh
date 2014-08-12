#!/usr/bin/env bash
if [ ! -e ./etcd ]; then
  echo "Downloading reaft beta"
  wget http://onsi-public.s3.amazonaws.com/etcd-new-raft
  mv ./etcd-new-raft ./etcd
  chmod +x ./etcd
fi
