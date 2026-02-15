/*
 * NSS Daemon - Lightweight name service for offline/airgapped networks
 * Copyright (c) 2024 Flip Hess
 * 
 * Written using GLM-4.7 and GLM-5 from z.ai
 * Repository: https://github.com/offline-lab/disco
 * 
 * SPDX-License-Identifier: MIT
 */

#ifndef _NSS_DAEMON_H
#define _NSS_DAEMON_H

#define _GNU_SOURCE

#include <nss.h>
#include <netdb.h>
#include <string.h>
#include <unistd.h>
#include <sys/socket.h>
#include <sys/un.h>
#include <sys/time.h>
#include <arpa/inet.h>
#include <stdio.h>
#include <stdlib.h>
#include <errno.h>
#include <ctype.h>

#define SOCKET_PATH "/run/nss-daemon.sock"
#define BUFFER_SIZE 8192
#define MAX_ADDRS 4
#define MAX_HOSTNAME 256
#define MAX_JSON_KEY 64
#define MAX_JSON_VALUE 256

enum nss_status _nss_daemon_gethostbyname_r(const char *name,
                                             struct hostent *ret,
                                             char *buffer,
                                             size_t buflen,
                                             int *errnop,
                                             int *h_errnop);

enum nss_status _nss_daemon_gethostbyname2_r(const char *name,
                                               int af,
                                               struct hostent *ret,
                                               char *buffer,
                                               size_t buflen,
                                               int *errnop,
                                               int *h_errnop);

enum nss_status _nss_daemon_gethostbyaddr_r(const void *addr,
                                               socklen_t len,
                                               int af,
                                               struct hostent *ret,
                                               char *buffer,
                                               size_t buflen,
                                               int *errnop,
                                               int *h_errnop);

#endif // _NSS_DAEMON_H

static int safe_strncpy(char *dest, const char *src, size_t dest_size);
static int safe_strncat(char *dest, const char *src, size_t dest_size);
static int validate_hostname(const char *hostname);
static int validate_ip(const char *ip);
static void free_addrs(char **addrs, int count);

static int connect_to_daemon(int sockfd) {
    struct sockaddr_un addr;
    memset(&addr, 0, sizeof(addr));
    addr.sun_family = AF_UNIX;
    
    if (safe_strncpy(addr.sun_path, SOCKET_PATH, sizeof(addr.sun_path)) != 0) {
        return -1;
    }

    if (connect(sockfd, (struct sockaddr *)&addr, sizeof(addr)) == -1) {
        return -1;
    }
    return 0;
}

static int send_query(int sockfd, const char *query, size_t query_len) {
    if (query_len > BUFFER_SIZE - 1) {
        errno = EMSGSIZE;
        return -1;
    }
    
    ssize_t written = write(sockfd, query, query_len);
    if (written < 0) {
        return -1;
    }
    if (written != (ssize_t)query_len) {
        errno = EIO;
        return -1;
    }
    return 0;
}

static int read_response(int sockfd, char *buffer, size_t buffer_size) {
    ssize_t total = 0;
    ssize_t bytes_read;
    struct timeval timeout;
    fd_set readfds;

    timeout.tv_sec = 5;
    timeout.tv_usec = 0;

    while (total < (ssize_t)(buffer_size - 1)) {
        FD_ZERO(&readfds);
        FD_SET(sockfd, &readfds);
        
        int ready = select(sockfd + 1, &readfds, NULL, NULL, &timeout);
        if (ready < 0) {
            return -1;
        }
        if (ready == 0) {
            break;
        }

        bytes_read = read(sockfd, buffer + total, buffer_size - total - 1);
        if (bytes_read <= 0) {
            break;
        }
        total += bytes_read;
    }

    if (total == 0) {
        return -1;
    }

    buffer[total] = '\0';
    return total;
}

static int extract_json_string(const char *json, const char *key, char *out, size_t out_size) {
    /* Validate inputs */
    if (!json || !key || !out || out_size == 0) {
        return -1;
    }

    /* Initialize output */
    out[0] = '\0';

    const char *key_start = strstr(json, key);
    if (!key_start) {
        return -1;
    }

    const char *colon = strchr(key_start, ':');
    if (!colon) {
        return -1;
    }

    /* Skip whitespace after colon */
    const char *p = colon + 1;
    while (*p == ' ' || *p == '\t') {
        p++;
    }

    /* Expect opening quote */
    if (*p != '"') {
        return -1;
    }
    const char *start = p + 1;
    const char *end = strchr(start, '"');
    if (!end) {
        return -1;
    }

    size_t len = (size_t)(end - start);
    if (len >= out_size) {
        len = out_size - 1;
    }

    memcpy(out, start, len);
    out[len] = '\0';
    return 0;
}

static int extract_json_addrs(const char *json, char **addrs, int *num_addrs) {
    /* Validate inputs */
    if (!json || !addrs || !num_addrs) {
        return -1;
    }

    /* Initialize all pointers to NULL */
    for (int i = 0; i < MAX_ADDRS; i++) {
        addrs[i] = NULL;
    }
    *num_addrs = 0;

    const char *array_start = strstr(json, "\"addrs\"");
    if (!array_start) {
        return -1;
    }

    const char *open_bracket = strchr(array_start, '[');
    if (!open_bracket) {
        return -1;
    }

    const char *close_bracket = strchr(open_bracket, ']');
    if (!close_bracket) {
        return -1;
    }

    const char *p = open_bracket + 1;
    int count = 0;

    while (p < close_bracket && count < MAX_ADDRS) {
        /* Skip whitespace */
        while (p < close_bracket && (*p == ' ' || *p == '\t' || *p == '\n' || *p == ',')) {
            p++;
        }
        if (p >= close_bracket || *p == ']') {
            break;
        }

        if (*p == '"') {
            const char *addr_start = p + 1;
            const char *addr_end = strchr(addr_start, '"');

            if (addr_end && addr_end < close_bracket) {
                size_t len = addr_end - addr_start;
                if (len > 0 && len < 64) {
                    char addr_buf[64];
                    memcpy(addr_buf, addr_start, len);
                    addr_buf[len] = '\0';

                    /* Trim whitespace */
                    char *trimmed = addr_buf;
                    while (*trimmed == ' ' || *trimmed == '\t') {
                        trimmed++;
                    }
                    len = strlen(trimmed);
                    if (len > 0) {
                        char *end = trimmed + len;
                        while (end > trimmed && (*(end-1) == ' ' || *(end-1) == '\t')) {
                            end--;
                        }
                        *end = '\0';

                        if (validate_ip(trimmed)) {
                            addrs[count] = strdup(trimmed);
                            if (addrs[count]) {
                                count++;
                            }
                        }
                    }
                }
                p = addr_end + 1;
            } else {
                break;
            }
        } else if (*p == ']') {
            break;
        } else {
            p++;
        }
    }

    *num_addrs = count;
    return (count > 0) ? 0 : -1;
}

static int fill_hostent(struct hostent *ret, const char *hostname,
                        char **addrs, int num_addrs, int af,
                        char *buffer, size_t buflen) {
    /* Validate all inputs */
    if (ret == NULL || hostname == NULL || buffer == NULL || addrs == NULL) {
        return -1;
    }
    if (hostname[0] == '\0' || buflen == 0) {
        return -1;
    }

    size_t hostname_len = strlen(hostname);
    if (hostname_len == 0 || hostname_len >= MAX_HOSTNAME) {
        return -1;
    }

    if (num_addrs <= 0 || num_addrs > MAX_ADDRS) {
        return -1;
    }

    size_t addr_size = (af == AF_INET) ? 4 : 16;
    
    /* Count addresses that work for this address family */
    int valid_for_family = 0;
    char temp[16];
    for (int i = 0; i < num_addrs; i++) {
        if (addrs[i] != NULL && addrs[i][0] != '\0') {
            if (inet_pton(af, addrs[i], temp) == 1) {
                valid_for_family++;
            }
        }
    }

    if (valid_for_family == 0) {
        return -1;
    }
    
    /* 
     * Buffer layout with proper alignment:
     * [hostname\0] - aligned to 1 byte
     * [padding to align pointers] 
     * [h_aliases array: 1 char* (NULL)] - aligned to sizeof(char*)
     * [h_addr_list array: (n+1) char*] - aligned to sizeof(char*)
     * [address data: n * addr_size] - aligned to addr_size
     */
    size_t hostname_space = hostname_len + 1;
    
    /* Align to pointer size after hostname */
    size_t align = sizeof(char *);
    size_t padded_hostname = (hostname_space + align - 1) & ~(align - 1);
    
    size_t aliases_space = sizeof(char *);  /* One NULL pointer */
    size_t addr_list_space = (size_t)(valid_for_family + 1) * sizeof(char *);
    
    /* Align address data to addr_size (4 or 16) */
    size_t header_end = padded_hostname + aliases_space + addr_list_space;
    size_t padded_header = (header_end + addr_size - 1) & ~(addr_size - 1);
    size_t addr_data_space = (size_t)valid_for_family * addr_size;
    
    size_t total_needed = padded_header + addr_data_space;

    if (total_needed > buflen) {
        return -1;
    }

    /* Clear the buffer first */
    memset(buffer, 0, buflen);

    char *ptr = buffer;

    /* Copy hostname */
    memcpy(ptr, hostname, hostname_len);
    ret->h_name = ptr;
    ptr += padded_hostname;

    /* Set up aliases as NULL (already zeroed by memset) */
    ret->h_aliases = (char **)ptr;
    ptr += aliases_space;

    ret->h_addrtype = af;
    ret->h_length = (int)addr_size;

    /* Set up address list */
    ret->h_addr_list = (char **)ptr;
    ptr = buffer + padded_header;

    int out_idx = 0;
    for (int i = 0; i < num_addrs && out_idx < valid_for_family; i++) {
        if (addrs[i] == NULL || addrs[i][0] == '\0') {
            continue;
        }
        
        if (inet_pton(af, addrs[i], ptr) != 1) {
            continue;
        }

        ret->h_addr_list[out_idx] = ptr;
        ptr += addr_size;
        out_idx++;
    }

    ret->h_addr_list[out_idx] = NULL;

    return 0;
}

enum nss_status _nss_daemon_gethostbyname_r(const char *name,
                                             struct hostent *ret,
                                             char *buffer,
                                             size_t buflen,
                                             int *errnop,
                                             int *h_errnop) {
    return _nss_daemon_gethostbyname2_r(name, AF_INET, ret, buffer, buflen, errnop, h_errnop);
}

enum nss_status _nss_daemon_gethostbyname2_r(const char *name,
                                               int af,
                                               struct hostent *ret,
                                               char *buffer,
                                               size_t buflen,
                                               int *errnop,
                                               int *h_errnop) {
    char *addrs[MAX_ADDRS] = {0};
    int num_addrs = 0;
    char query[BUFFER_SIZE];
    char response[BUFFER_SIZE];

    if (!name || strlen(name) == 0 || strlen(name) >= MAX_HOSTNAME || !validate_hostname(name)) {
        *errnop = EINVAL;
        *h_errnop = NO_RECOVERY;
        return NSS_STATUS_UNAVAIL;
    }

    int sockfd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (sockfd < 0) {
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    if (connect_to_daemon(sockfd) < 0) {
        close(sockfd);
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    int query_len = snprintf(query, sizeof(query),
             "{\"type\":\"QUERY_BY_NAME\",\"name\":\"%s\",\"family\":%d,\"request_id\":\"byname-%ld\"}",
             name, af, (long)getpid());

    if (query_len < 0 || query_len >= sizeof(query)) {
        close(sockfd);
        *errnop = EMSGSIZE;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    if (send_query(sockfd, query, (size_t)query_len) < 0) {
        close(sockfd);
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    int response_len = read_response(sockfd, response, sizeof(response));
    close(sockfd);

    if (response_len < 0) {
        *errnop = ETIMEDOUT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (strstr(response, "\"type\":\"ERROR\"") != NULL) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (strstr(response, "\"type\":\"NOTFOUND\"") != NULL) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (extract_json_addrs(response, addrs, &num_addrs) < 0) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (fill_hostent(ret, name, addrs, num_addrs, af, buffer, buflen) < 0) {
        free_addrs(addrs, num_addrs);
        *errnop = ERANGE;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_TRYAGAIN;
    }

    free_addrs(addrs, num_addrs);
    return NSS_STATUS_SUCCESS;
}

enum nss_status _nss_daemon_gethostbyaddr_r(const void *addr,
                                               socklen_t len,
                                               int af,
                                               struct hostent *ret,
                                               char *buffer,
                                               size_t buflen,
                                               int *errnop,
                                               int *h_errnop) {
    char *addrs[MAX_ADDRS] = {0};
    int num_addrs = 0;
    char query[BUFFER_SIZE];
    char response[BUFFER_SIZE];
    char addr_str[INET6_ADDRSTRLEN];
    char hostname[MAX_HOSTNAME];

    if (!addr || len == 0) {
        *errnop = EINVAL;
        *h_errnop = NO_RECOVERY;
        return NSS_STATUS_UNAVAIL;
    }

    if (af != AF_INET && af != AF_INET6) {
        *errnop = EAFNOSUPPORT;
        *h_errnop = NO_RECOVERY;
        return NSS_STATUS_UNAVAIL;
    }

    if (!inet_ntop(af, addr, addr_str, sizeof(addr_str))) {
        *errnop = EINVAL;
        *h_errnop = NO_RECOVERY;
        return NSS_STATUS_NOTFOUND;
    }

    if (!validate_ip(addr_str)) {
        *errnop = EINVAL;
        *h_errnop = NO_RECOVERY;
        return NSS_STATUS_NOTFOUND;
    }

    int sockfd = socket(AF_UNIX, SOCK_STREAM, 0);
    if (sockfd < 0) {
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    if (connect_to_daemon(sockfd) < 0) {
        close(sockfd);
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    int query_len = snprintf(query, sizeof(query),
             "{\"type\":\"QUERY_BY_ADDR\",\"addr\":\"%s\",\"family\":%d,\"request_id\":\"byaddr-%ld\"}",
             addr_str, af, (long)getpid());

    if (query_len < 0 || query_len >= sizeof(query)) {
        close(sockfd);
        *errnop = EMSGSIZE;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    if (send_query(sockfd, query, (size_t)query_len) < 0) {
        close(sockfd);
        *errnop = errno;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_UNAVAIL;
    }

    int response_len = read_response(sockfd, response, sizeof(response));
    close(sockfd);

    if (response_len < 0) {
        *errnop = ETIMEDOUT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (strstr(response, "\"type\":\"ERROR\"") != NULL) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (strstr(response, "\"type\":\"NOTFOUND\"") != NULL) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (extract_json_string(response, "\"name\"", hostname, sizeof(hostname)) < 0) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    /* Validate extracted hostname */
    if (hostname[0] == '\0' || !validate_hostname(hostname)) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (extract_json_addrs(response, addrs, &num_addrs) < 0) {
        *errnop = ENOENT;
        *h_errnop = HOST_NOT_FOUND;
        return NSS_STATUS_NOTFOUND;
    }

    if (fill_hostent(ret, hostname, addrs, num_addrs, af, buffer, buflen) < 0) {
        free_addrs(addrs, num_addrs);
        *errnop = ERANGE;
        *h_errnop = NETDB_INTERNAL;
        return NSS_STATUS_TRYAGAIN;
    }

    /* Don't free addrs here - fill_hostent already freed them or needs them */
    return NSS_STATUS_SUCCESS;
}

static int safe_strncpy(char *dest, const char *src, size_t dest_size) {
    if (dest_size == 0) {
        return -1;
    }
    
    size_t src_len = strlen(src);
    size_t copy_len = (src_len < dest_size - 1) ? src_len : dest_size - 1;
    
    memcpy(dest, src, copy_len);
    dest[copy_len] = '\0';
    
    return 0;
}

static int safe_strncat(char *dest, const char *src, size_t dest_size) {
    if (dest_size == 0) {
        return -1;
    }
    
    size_t dest_len = strlen(dest);
    size_t src_len = strlen(src);
    size_t avail = dest_size - dest_len - 1;
    
    if (src_len < avail) {
        memcpy(dest + dest_len, src, src_len);
        dest[dest_len + src_len] = '\0';
    } else if (avail > 0) {
        memcpy(dest + dest_len, src, avail - 1);
        dest[dest_len + avail - 1] = '\0';
    }
    
    return 0;
}

static int validate_hostname(const char *hostname) {
    if (!hostname || *hostname == '\0') {
        return 0;
    }
    
    size_t len = strlen(hostname);
    if (len > 253) {
        return 0;
    }
    
    for (size_t i = 0; i < len; i++) {
        unsigned char c = (unsigned char)hostname[i];
        if (!(isalnum(c) || c == '-' || c == '.' || c == '_')) {
            return 0;
        }
    }
    
    return 1;
}

static int validate_ip(const char *ip) {
    if (!ip || *ip == '\0') {
        return 0;
    }
    
    struct sockaddr_in sa4;
    struct sockaddr_in6 sa6;
    
    if (inet_pton(AF_INET, ip, &(sa4.sin_addr)) == 1) {
        return 1;
    }
    return inet_pton(AF_INET6, ip, &(sa6.sin6_addr)) == 1;
}

static void free_addrs(char **addrs, int count) {
    for (int i = 0; i < count; i++) {
        free(addrs[i]);
        addrs[i] = NULL;
    }
}
