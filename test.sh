#!/bin/bash

docker run -itd --cap-add=NET_ADMIN --net="none" alpine:latest
