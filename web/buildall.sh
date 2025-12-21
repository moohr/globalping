#!/bin/bash

docker build \
  --push \
  --tag ghcr.io/internetworklab/globalping-web:latest .
