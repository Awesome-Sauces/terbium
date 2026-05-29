#!/usr/bin/env bash
set -e

go run gen.go

clang -x ir -c main.ll -o main.o

nvcc -std=c++20 -arch=sm_75 -c matmul_generated.cu -o matmul_generated.o

nvcc main.o matmul_generated.o -o matmul_cuda_app

echo "Built: ./matmul_cuda_app"