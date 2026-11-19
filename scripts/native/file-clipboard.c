/* RDP file clipboard extension for the pinned Guacamole/FreeRDP build.
 * The source writes a bounded snapshot. The web client only sees a capability
 * for it. The destination opens it inside the authenticated owner's directory.
 * Included by cliprdr.c. Source state belongs to the CLIPRDR callback thread. */
#include <fcntl.h>
#include <sys/random.h>
#include <sys/stat.h>
#include <unistd.h>
#include <freerdp/utils/cliprdr_utils.h>

#define FILE_MIME "application/x-fnos-rdp-files"
#define FILE_LIMIT (256U * 1024 * 1024)
#define FILE_COUNT 64
#define DESCRIPTOR_SIZE 592
#define RANGE_SIZE (128U * 1024)
#define FILE_FORMAT 0xc001

typedef struct {
    FILE* fp;
    BYTE* descriptors;
    UINT32 length, count;
    UINT32 offsets[FILE_COUNT + 1];
} file_snapshot;

typedef struct file_clipboard {
    file_snapshot source, target;
    char* source_path;
    UINT32 index, position, stream_id, requested;
    UINT32 file_format, queued_format;
    int format_pending, superseded;
    pthread_mutex_t target_lock;
    UINT32 sent_lists, received_lists, waiting_list;
    char waiting[99];
} file_clipboard;

static void snapshot_close(file_snapshot* s) {
    if (s->fp) fclose(s->fp);
    free(s->descriptors);
    memset(s, 0, sizeof(*s));
}

static void source_clear(file_clipboard* f) {
    snapshot_close(&f->source);
    if (f->source_path) unlink(f->source_path);
    free(f->source_path);
    f->source_path = NULL;
    f->index = f->position = f->requested = 0;
}

static void file_event(guac_rdp_clipboard* cb, const char* event) {
    guac_common_clipboard* msg = guac_common_clipboard_alloc();
    guac_common_clipboard_reset(msg, FILE_MIME);
    guac_common_clipboard_append(msg, event, strlen(event));
    guac_common_clipboard_send(msg, cb->client);
    guac_common_clipboard_free(msg);
}

static const char* file_directory(guac_rdp_clipboard* cb) {
    guac_rdp_client* rdp = cb->client->data;
    return rdp->settings->file_clipboard_path;
}

static UINT file_failure(guac_rdp_clipboard* cb) {
    source_clear(cb->files);
    file_event(cb, "failed");
    return CHANNEL_RC_OK;
}

/* Retain only regular files with a plain basename. Validate the original
 * wire bytes too, as the FreeRDP parser normalizes some malformed names. */
static int snapshot_descriptors(file_snapshot* s, const BYTE* data, UINT32 length) {
    if (!data || length < 4 || length > 4 + FILE_COUNT * DESCRIPTOR_SIZE) return 0;
    UINT32 count = data[0];
    if (!count || count > FILE_COUNT || data[1] || data[2] || data[3] ||
            length != 4 + count * DESCRIPTOR_SIZE) return 0;
    FILEDESCRIPTORW* list = NULL;
    UINT32 parsed = 0;
    if (cliprdr_parse_file_list(data, length, &list, &parsed) || parsed != count) {
        free(list);
        return 0;
    }
    UINT32 total = length;
    for (UINT32 i = 0; i < count; i++) {
        const FILEDESCRIPTORW* d = &list[i];
        const BYTE* name = data + 4 + i * DESCRIPTOR_SIZE + 72;
        size_t n = 0;
        for (; n < 260; n++) {
            UINT16 ch = name[n*2] | (name[n*2+1] << 8);
            if (!ch) break;
            if (ch < 32 || (ch < 128 && strchr("/\\:*?\"<>|", ch))) goto invalid;
        }
        if (!n || n == 260 || (!name[2*n-1] &&
                (name[2*n-2] == '.' || name[2*n-2] == ' '))) goto invalid;
        if (!(d->dwFlags & FD_FILESIZE) || d->nFileSizeHigh ||
                (d->dwFileAttributes & (FILE_ATTRIBUTE_DIRECTORY | FILE_ATTRIBUTE_REPARSE_POINT)) ||
                d->nFileSizeLow > FILE_LIMIT - (total - length)) goto invalid;
        s->offsets[i] = total;
        total += d->nFileSizeLow;
    }
    s->offsets[count] = total;
    s->descriptors = malloc(length);
    if (!s->descriptors) goto invalid;
    memcpy(s->descriptors, data, length);
    s->length = length;
    s->count = count;
    free(list);
    return 1;
invalid:
    free(list);
    return 0;
}

static UINT file_next(CliprdrClientContext* ctx) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    file_snapshot* s = &f->source;
    while (f->index < s->count) {
        UINT32 size = s->offsets[f->index + 1] - s->offsets[f->index];
        if (f->position < size) {
            UINT32 left = size - f->position;
            f->requested = left < RANGE_SIZE ? left : RANGE_SIZE;
            CLIPRDR_FILE_CONTENTS_REQUEST request = {
                .streamId = ++f->stream_id, .listIndex = f->index,
                .dwFlags = FILECONTENTS_RANGE, .nPositionLow = f->position,
                .cbRequested = f->requested
            };
            if (ctx->ClientFileContentsRequest(ctx, &request)) return file_failure(cb);
            return CHANNEL_RC_OK;
        }
        f->index++;
        f->position = 0;
    }
    int error = fclose(s->fp);
    s->fp = NULL;
    if (error) return file_failure(cb);
    file_event(cb, strrchr(f->source_path, '/') + 1);
    return CHANNEL_RC_OK;
}

static UINT source_descriptors(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_DATA_RESPONSE* r) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    file_snapshot* s = &f->source;
    if (!(r->common.msgFlags & CB_RESPONSE_OK) ||
            !snapshot_descriptors(s, r->requestedFormatData, r->common.dataLen))
        return file_failure(cb);
    BYTE random[16];
    if (getrandom(random, sizeof(random), 0) != sizeof(random)) return file_failure(cb);
    char token[33];
    for (size_t i = 0; i < sizeof(random); i++) snprintf(token + 2*i, 3, "%02x", random[i]);
    size_t capacity = strlen(file_directory(cb)) + sizeof(token) + 1;
    char* path = malloc(capacity);
    if (!path) return file_failure(cb);
    snprintf(path, capacity, "%s/%s", file_directory(cb), token);
    int fd = open(path, O_CREAT | O_EXCL | O_WRONLY | O_CLOEXEC, 0600);
    if (fd < 0) { free(path); return file_failure(cb); }
    f->source_path = path;
    s->fp = fdopen(fd, "wb");
    if (!s->fp) { close(fd); return file_failure(cb); }
    if (fwrite(s->descriptors, 1, s->length, s->fp) != s->length) return file_failure(cb);
    return file_next(ctx);
}

static UINT file_received(CliprdrClientContext* ctx, const CLIPRDR_FILE_CONTENTS_RESPONSE* r) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    if (!f->source.fp || r->streamId != f->stream_id) return CHANNEL_RC_OK;
    if (!(r->common.msgFlags & CB_RESPONSE_OK) || !r->cbRequested ||
            r->cbRequested > f->requested || !r->requestedData ||
            fwrite(r->requestedData, 1, r->cbRequested, f->source.fp) != r->cbRequested)
        return file_failure(cb);
    f->position += r->cbRequested;
    return file_next(ctx);
}

static int open_snapshot(const char* dir, const char* token, file_snapshot* s) {
    /* Both directory components are generated by the gateway. No client
     * path, symlink, absolute name or cross-owner lookup is permitted. */
    if (strlen(token) != 97 || token[64] != '/' ||
            strspn(token, "0123456789abcdef") != 64 ||
            strspn(token + 65, "0123456789abcdef") != 32) return 0;
    char* parent = strdup(dir);
    if (!parent) return 0;
    char* slash = strrchr(parent, '/');
    if (!slash) { free(parent); return 0; }
    *slash = 0;
    int owner = open(parent, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    free(parent);
    if (owner < 0) return 0;
    char session[65];
    memcpy(session, token, 64);
    session[64] = 0;
    int src = openat(owner, session, O_RDONLY | O_DIRECTORY | O_NOFOLLOW | O_CLOEXEC);
    close(owner);
    if (src < 0) return 0;
    int fd = openat(src, token + 65, O_RDONLY | O_NOFOLLOW | O_CLOEXEC | O_NONBLOCK);
    close(src);
    if (fd < 0) return 0;
    struct stat st;
    if (fstat(fd, &st) || !S_ISREG(st.st_mode)) { close(fd); return 0; }
    s->fp = fdopen(fd, "rb");
    if (!s->fp) { close(fd); return 0; }
    BYTE header[4 + FILE_COUNT * DESCRIPTOR_SIZE];
    if (fread(header, 1, 4, s->fp) != 4 || header[0] > FILE_COUNT) return 0;
    UINT32 bytes = header[0] * DESCRIPTOR_SIZE;
    return fread(header + 4, 1, bytes, s->fp) == bytes &&
        snapshot_descriptors(s, header, bytes + 4) && st.st_size == s->offsets[s->count];
}

static void file_install(guac_rdp_clipboard* cb, const char* token) {
    file_clipboard* f = cb->files;
    file_snapshot s = {0};
    if (!cb->cliprdr || !open_snapshot(file_directory(cb), token, &s)) {
        snapshot_close(&s);
        file_event(cb, "rejected");
        return;
    }
    guac_rdp_client* rdp = cb->client->data;
    pthread_mutex_lock(&rdp->message_lock);
    pthread_mutex_lock(&f->target_lock);
    snapshot_close(&f->target);
    f->target = s;
    f->waiting_list = ++f->sent_lists;
    snprintf(f->waiting, sizeof(f->waiting), "%s", token);
    pthread_mutex_unlock(&f->target_lock);
    CLIPRDR_FORMAT format = { .formatId = FILE_FORMAT, .formatName = "FileGroupDescriptorW" };
    CLIPRDR_FORMAT_LIST formats = { .common.msgType = CB_FORMAT_LIST, .numFormats = 1, .formats = &format };
    UINT result = cb->cliprdr ? cb->cliprdr->ClientFormatList(cb->cliprdr, &formats) : 1;
    pthread_mutex_unlock(&rdp->message_lock);
    if (result) file_event(cb, "rejected");
}

static UINT file_list_ack(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_LIST_RESPONSE* response) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    char event[110] = "";
    pthread_mutex_lock(&f->target_lock);
    if (++f->received_lists == f->waiting_list && *f->waiting) {
        if (response->common.msgFlags & CB_RESPONSE_OK)
            snprintf(event, sizeof(event), "installed:%s", f->waiting);
        else strcpy(event, "rejected");
        *f->waiting = 0;
    }
    pthread_mutex_unlock(&f->target_lock);
    if (*event) file_event(cb, event);
    return CHANNEL_RC_OK;
}

static void target_clear(file_clipboard* f, int sending_list) {
    pthread_mutex_lock(&f->target_lock);
    if (sending_list) f->sent_lists++;
    snapshot_close(&f->target);
    *f->waiting = 0;
    pthread_mutex_unlock(&f->target_lock);
}

static UINT file_descriptors_requested(CliprdrClientContext* ctx) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    pthread_mutex_lock(&f->target_lock);
    CLIPRDR_FORMAT_DATA_RESPONSE r = {
        .common.msgFlags = f->target.fp ? CB_RESPONSE_OK : CB_RESPONSE_FAIL,
        .common.dataLen = f->target.length, .requestedFormatData = f->target.descriptors
    };
    UINT result = ctx->ClientFormatDataResponse(ctx, &r);
    pthread_mutex_unlock(&f->target_lock);
    return result;
}

static UINT file_requested(CliprdrClientContext* ctx, const CLIPRDR_FILE_CONTENTS_REQUEST* req) {
    guac_rdp_clipboard* cb = ctx->custom;
    file_clipboard* f = cb->files;
    file_snapshot* s = &f->target;
    CLIPRDR_FILE_CONTENTS_RESPONSE r = { .common.msgFlags = CB_RESPONSE_FAIL, .streamId = req->streamId };
    BYTE size_bytes[8] = {0};
    BYTE* buffer = NULL;
    pthread_mutex_lock(&f->target_lock);
    if (s->fp && req->listIndex < s->count) {
        UINT32 start = s->offsets[req->listIndex], size = s->offsets[req->listIndex+1] - start;
        if (req->dwFlags == FILECONTENTS_SIZE) {
            for (int i = 0; i < 4; i++) size_bytes[i] = size >> (8*i);
            r.requestedData = size_bytes;
            r.cbRequested = 8;
            r.common.msgFlags = CB_RESPONSE_OK;
        }
        else if (req->dwFlags == FILECONTENTS_RANGE && !req->nPositionHigh &&
                req->nPositionLow <= size && req->cbRequested <= 4U*1024*1024) {
            UINT32 left = size - req->nPositionLow;
            UINT32 n = req->cbRequested < left ? req->cbRequested : left;
            buffer = malloc(n ? n : 1);
            if (buffer && pread(fileno(s->fp), buffer, n, start + req->nPositionLow) == n) {
                r.requestedData = buffer;
                r.cbRequested = n;
                r.common.msgFlags = CB_RESPONSE_OK;
            }
        }
    }
    pthread_mutex_unlock(&f->target_lock);
    UINT result = ctx->ClientFileContentsResponse(ctx, &r);
    free(buffer);
    return result;
}

static int file_token_blob(guac_user* user, guac_stream* stream, void* data, int length) {
    guac_rdp_client* rdp = user->client->data;
    stream->blob_handler = NULL;
    if (length != 97 || memchr(data, 0, length)) return 0;
    char token[98];
    memcpy(token, data, length);
    token[length] = 0;
    file_install(rdp->clipboard, token);
    return 0;
}
