/*
 * Compile-only ABI probe for Android arm64/API 29+.
 * Keep this independent from Go so NDK header drift fails before a device run.
 */
#include <stddef.h>
#include <dlfcn.h>
#include <pthread.h>
#include <signal.h>

_Static_assert(sizeof(sigset_t) == 8, "Bionic LP64 sigset_t must be one word");
_Static_assert(_Alignof(sigset_t) == 8, "Bionic LP64 sigset_t alignment changed");
_Static_assert(sizeof(pthread_attr_t) == 56, "Bionic LP64 pthread_attr_t changed");
_Static_assert(_Alignof(pthread_attr_t) == 8, "Bionic LP64 pthread_attr_t alignment changed");
_Static_assert(sizeof(pthread_cond_t) == 48, "Bionic LP64 pthread_cond_t changed");
_Static_assert(_Alignof(pthread_cond_t) == 4, "Bionic LP64 pthread_cond_t alignment changed");
_Static_assert(sizeof(pthread_mutex_t) == 40, "Bionic LP64 pthread_mutex_t changed");
_Static_assert(_Alignof(pthread_mutex_t) == 4, "Bionic LP64 pthread_mutex_t alignment changed");
_Static_assert(sizeof(pthread_t) == 8, "Bionic LP64 pthread_t changed");
_Static_assert(sizeof(pthread_key_t) == 4, "Bionic pthread_key_t changed");
_Static_assert(sizeof(stack_t) == 24, "Bionic LP64 stack_t changed");
_Static_assert(offsetof(stack_t, ss_size) == 16, "Bionic stack_t layout changed");
_Static_assert(RTLD_NOW == 2, "Bionic RTLD_NOW changed");
_Static_assert(RTLD_LOCAL == 0, "Bionic RTLD_LOCAL changed");
_Static_assert(RTLD_NODELETE == 0x1000, "Bionic RTLD_NODELETE changed");

int main(void) {
	return pthread_detach((pthread_t)0) == 0;
}
