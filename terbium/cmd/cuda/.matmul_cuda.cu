#include <cuda_runtime.h>

#include <cstdint>
#include <cstdio>
#include <cstdlib>

#define CUDA_CHECK(call)                                                     \
    do {                                                                     \
        cudaError_t err__ = (call);                                          \
        if (err__ != cudaSuccess) {                                          \
            std::fprintf(stderr,                                             \
                         "CUDA error at %s:%d: %s\n",                       \
                         __FILE__,                                           \
                         __LINE__,                                           \
                         cudaGetErrorString(err__));                         \
            std::exit(1);                                                    \
        }                                                                    \
    } while (0)

__global__ void matmul2x2_kernel(const int32_t* A,
                                 const int32_t* B,
                                 int32_t* C) {
    int row = threadIdx.y;
    int col = threadIdx.x;

    if (row < 2 && col < 2) {
        int idx = row * 2 + col;

        C[idx] =
            A[row * 2 + 0] * B[0 * 2 + col] +
            A[row * 2 + 1] * B[1 * 2 + col];
    }
}

// This symbol name must exactly match the LLVM declaration:
//
// declare void @matmul2x2([4 x i32]*, [4 x i32]*, [4 x i32]*)
//
// LLVM [4 x i32]* maps naturally to C/C++:
//
// int32_t (*a)[4]
//
extern "C" void matmul2x2(int32_t (*a)[4],
                          int32_t (*b)[4],
                          int32_t (*c)[4]) {
    constexpr std::size_t bytes = 4 * sizeof(int32_t);

    int32_t* dA = nullptr;
    int32_t* dB = nullptr;
    int32_t* dC = nullptr;

    CUDA_CHECK(cudaMalloc(reinterpret_cast<void**>(&dA), bytes));
    CUDA_CHECK(cudaMalloc(reinterpret_cast<void**>(&dB), bytes));
    CUDA_CHECK(cudaMalloc(reinterpret_cast<void**>(&dC), bytes));

    CUDA_CHECK(cudaMemcpy(dA, *a, bytes, cudaMemcpyHostToDevice));
    CUDA_CHECK(cudaMemcpy(dB, *b, bytes, cudaMemcpyHostToDevice));

    dim3 block(2, 2);
    dim3 grid(1);

    matmul2x2_kernel<<<grid, block>>>(dA, dB, dC);

    CUDA_CHECK(cudaGetLastError());
    CUDA_CHECK(cudaDeviceSynchronize());

    CUDA_CHECK(cudaMemcpy(*c, dC, bytes, cudaMemcpyDeviceToHost));

    CUDA_CHECK(cudaFree(dA));
    CUDA_CHECK(cudaFree(dB));
    CUDA_CHECK(cudaFree(dC));
}