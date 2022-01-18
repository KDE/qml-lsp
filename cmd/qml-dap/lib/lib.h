#pragma once

#ifdef __cplusplus
extern "C" {
#endif

typedef struct Handle Handle;

typedef void (*JSONNotification)(void* userData, const char* data);

Handle* makeLibraryHandle(void* userData, void* callback);
void execHandle(Handle* handle);
const char* invokeHandle(Handle* handle, const char* op);
void freeLibraryHandle(Handle* handle);

#ifdef __cplusplus
}
#endif