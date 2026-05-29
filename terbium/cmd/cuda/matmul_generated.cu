#include <cuda_runtime.h>
#include <stdint.h>
#include <stdio.h>
#include <stdlib.h>

#define CUDA_CHECK(call)                                                   \
    do {                                                                   \
        cudaError_t err__ = (call);                                        \
        if (err__ != cudaSuccess) {                                        \
            fprintf(stderr,                                                \
                    "CUDA error at %s:%d: %s\n",                          \
                    __FILE__,                                              \
                    __LINE__,                                              \
                    cudaGetErrorString(err__));                            \
            exit(1);                                                       \
        }                                                                  \
    } while (0)

__global__ void matmul2x2_kernel(const int32_t* A, const int32_t* B, int32_t* C) {
    int row = threadIdx.y + blockIdx.y * blockDim.y;
    int col = threadIdx.x + blockIdx.x * blockDim.x;

    if (row < 2 && col < 2) {
        int32_t sum = 0;
        for (int k = 0; k < 2; k++) {
            sum += A[row * 2 + k] * B[k * 2 + col];
        }
        C[row * 2 + col] = sum;
    }
}

extern "C" void matmul2x2(int32_t (*a)[4], int32_t (*b)[4], int32_t (*c)[4]) {
    const size_t bytes = 4 * sizeof(int32_t);

    int32_t* dA = NULL;
    int32_t* dB = NULL;
    int32_t* dC = NULL;

    CUDA_CHECK(cudaMalloc((void**)&dA, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dB, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dC, bytes));

    CUDA_CHECK(cudaMemcpy(dA, *a, bytes, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(dB, *b, bytes, cudaMemcpyHostToDevice));

    dim3 block(16, 16);
    dim3 grid((2 + block.x - 1) / block.x, (2 + block.y - 1) / block.y);

    matmul2x2_kernel<<<grid, block>>>(dA, dB, dC);

    CUDA_CHECK(cudaGetLastError());
    CUDA_CHECK(cudaDeviceSynchronize());

    CUDA_CHECK(cudaMemcpy(*c, dC, bytes, cudaMemcpyDeviceToHost));

    CUDA_CHECK(cudaFree(dA));
    CUDA_CHECK(cudaFree(dB));
    CUDA_CHECK(cudaFree(dC));
}

__global__ void matmul4x4_kernel(const int32_t* A, const int32_t* B, int32_t* C) {
    int row = threadIdx.y + blockIdx.y * blockDim.y;
    int col = threadIdx.x + blockIdx.x * blockDim.x;

    if (row < 4 && col < 4) {
        int32_t sum = 0;
        for (int k = 0; k < 4; k++) {
            sum += A[row * 4 + k] * B[k * 4 + col];
        }
        C[row * 4 + col] = sum;
    }
}

extern "C" void matmul4x4(int32_t (*a)[16], int32_t (*b)[16], int32_t (*c)[16]) {
    const size_t bytes = 16 * sizeof(int32_t);

    int32_t* dA = NULL;
    int32_t* dB = NULL;
    int32_t* dC = NULL;

    CUDA_CHECK(cudaMalloc((void**)&dA, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dB, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dC, bytes));

    CUDA_CHECK(cudaMemcpy(dA, *a, bytes, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(dB, *b, bytes, cudaMemcpyHostToDevice));

    dim3 block(16, 16);
    dim3 grid((4 + block.x - 1) / block.x, (4 + block.y - 1) / block.y);

    matmul4x4_kernel<<<grid, block>>>(dA, dB, dC);

    CUDA_CHECK(cudaGetLastError());
    CUDA_CHECK(cudaDeviceSynchronize());

    CUDA_CHECK(cudaMemcpy(*c, dC, bytes, cudaMemcpyDeviceToHost));

    CUDA_CHECK(cudaFree(dA));
    CUDA_CHECK(cudaFree(dB));
    CUDA_CHECK(cudaFree(dC));
}

__global__ void matmul8x8_kernel(const int32_t* A, const int32_t* B, int32_t* C) {
    int row = threadIdx.y + blockIdx.y * blockDim.y;
    int col = threadIdx.x + blockIdx.x * blockDim.x;

    if (row < 8 && col < 8) {
        int32_t sum = 0;
        for (int k = 0; k < 8; k++) {
            sum += A[row * 8 + k] * B[k * 8 + col];
        }
        C[row * 8 + col] = sum;
    }
}

extern "C" void matmul8x8(int32_t (*a)[64], int32_t (*b)[64], int32_t (*c)[64]) {
    const size_t bytes = 64 * sizeof(int32_t);

    int32_t* dA = NULL;
    int32_t* dB = NULL;
    int32_t* dC = NULL;

    CUDA_CHECK(cudaMalloc((void**)&dA, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dB, bytes));
    CUDA_CHECK(cudaMalloc((void**)&dC, bytes));

    CUDA_CHECK(cudaMemcpy(dA, *a, bytes, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(dB, *b, bytes, cudaMemcpyHostToDevice));

    dim3 block(16, 16);
    dim3 grid((8 + block.x - 1) / block.x, (8 + block.y - 1) / block.y);

    matmul8x8_kernel<<<grid, block>>>(dA, dB, dC);

    CUDA_CHECK(cudaGetLastError());
    CUDA_CHECK(cudaDeviceSynchronize());

    CUDA_CHECK(cudaMemcpy(*c, dC, bytes, cudaMemcpyDeviceToHost));

    CUDA_CHECK(cudaFree(dA));
    CUDA_CHECK(cudaFree(dB));
    CUDA_CHECK(cudaFree(dC));
}

