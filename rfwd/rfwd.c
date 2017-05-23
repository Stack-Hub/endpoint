#define _GNU_SOURCE
#include <stdio.h>
#include <dlfcn.h>
#include <string.h>
#include <stdlib.h>

// gcc -Wall -shared  -ldl -Wl,-soname,rfwd -o rfwd.so rfwd.c

typedef ssize_t write_fp(int fd, const void *buf, size_t count);
static write_fp *real_write;

void myinit(void) __attribute__((constructor));
void myinit(void)
{
    void *dl;
    if ((dl=dlopen(NULL,RTLD_NOW))) {
        real_write=dlsym(RTLD_NEXT,"write");
        if (!real_write) printf("error: %s\n",dlerror());
    } else {
        printf("dlopen() failed\n");
    }
}

ssize_t write(int fd, const void *buf, size_t count)
{
     // debug1: Remote connections from 192.168.0.1:0 forwarded to local address 127.0.0.1:1000
     //  Allocated port 44284 for remote forward to 127.0.0.1:1000
     // debug1: All remote forwarding requests processed
     if ( (fd==2) && (!strncmp(buf,"Allocated port ",15)) ) {
         char envbuf1[256],envbuf2[256];
         unsigned int rport;
         char lspec[256];
         int rc;

         rc=sscanf(buf,"Allocated port %u for remote forward to %256s",
          &rport,lspec);

         if ( rc==2 ) {
             snprintf(envbuf1,sizeof(envbuf1),"SSH_RFWD");
             snprintf(envbuf2,sizeof(envbuf2),"%u",rport);
             setenv(envbuf1,envbuf2,1);
         }
     }
     return real_write(fd,buf,count);
}