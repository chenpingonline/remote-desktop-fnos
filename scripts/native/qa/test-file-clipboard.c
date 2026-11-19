/* Exercise both CLIPRDR endpoints, with real FreeRDP file descriptors and
 * real snapshot files. Only the peer callbacks and the browser are simulated. */
#define guac_common_clipboard_send observe_message
#include "channels/cliprdr.c"
#undef guac_common_clipboard_send

#include <assert.h>
#include <sys/stat.h>

static char message[160];
static CLIPRDR_FILE_CONTENTS_REQUEST next_request;
static BYTE original[300001];
static const BYTE* expected;
static UINT32 expected_length;
static BYTE descriptors[4 + 2 * DESCRIPTOR_SIZE];

void observe_message(guac_common_clipboard* msg, guac_client* client) {
    assert(!strcmp(msg->mimetype, FILE_MIME));
    assert((size_t)msg->length < sizeof(message));
    memcpy(message, msg->buffer, msg->length);
    message[msg->length] = 0;
}
static UINT ask_format(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_DATA_REQUEST* r) {
    assert(r->requestedFormatId == 0xc013);
    return 0;
}
static UINT ask_range(CliprdrClientContext* ctx, const CLIPRDR_FILE_CONTENTS_REQUEST* r) {
    next_request = *r;
    return 0;
}
static UINT accept_formats(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_LIST_RESPONSE* r) {
    return 0;
}
static UINT offer_files(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_LIST* r) {
    assert(r->numFormats == 1 && !strcmp(r->formats[0].formatName, "FileGroupDescriptorW"));
    return 0;
}
static UINT verify_descriptors(CliprdrClientContext* ctx, const CLIPRDR_FORMAT_DATA_RESPONSE* r) {
    assert(r->common.msgFlags == CB_RESPONSE_OK && r->common.dataLen == sizeof(descriptors));
    assert(!memcmp(r->requestedFormatData, descriptors, sizeof(descriptors)));
    return 0;
}
static UINT verify_range(CliprdrClientContext* ctx, const CLIPRDR_FILE_CONTENTS_RESPONSE* r) {
    assert(r->streamId == 728);
    assert(r->common.msgFlags == (expected ? CB_RESPONSE_OK : CB_RESPONSE_FAIL));
    assert(r->cbRequested == expected_length);
    if (expected_length) assert(!memcmp(r->requestedData, expected, expected_length));
    return 0;
}
typedef struct {
    guac_rdp_settings settings;
    guac_rdp_client rdp;
    CliprdrClientContext ctx;
} endpoint;
static void setup(endpoint* ep, char* dir) {
    guac_client* client = guac_client_alloc();
    client->data = &ep->rdp;
    ep->rdp.settings = &ep->settings;
    ep->settings.file_clipboard_path = dir;
    ep->settings.disable_copy = ep->settings.disable_paste = 1; /* file-only mode */
    pthread_mutex_init(&ep->rdp.message_lock, NULL);
    ep->rdp.clipboard = guac_rdp_clipboard_alloc(client);
    ep->rdp.clipboard->cliprdr = &ep->ctx;
    ep->ctx.custom = ep->rdp.clipboard;
    ep->ctx.ClientFormatListResponse = accept_formats;
    ep->ctx.ClientFormatDataRequest = ask_format;
    ep->ctx.ClientFileContentsRequest = ask_range;
    ep->ctx.ClientFormatList = offer_files;
    ep->ctx.ClientFormatDataResponse = verify_descriptors;
    ep->ctx.ClientFileContentsResponse = verify_range;
}
int main(void) {
    char root[] = "/tmp/rdp-test-XXXXXX";
    assert(mkdtemp(root));
    char src[160], dst[160], other[160], token[100];
    char sid[65];
    memset(sid, 'a', 64); sid[64] = 0;
    snprintf(src, sizeof(src), "%s/%s", root, sid);
    snprintf(dst, sizeof(dst), "%s/target", root);
    snprintf(other, sizeof(other), "%s/other/target", root);
    assert(!mkdir(src, 0700) && !mkdir(dst, 0700));
    endpoint a = {0}, b = {0};
    setup(&a, src); setup(&b, dst);
    for (size_t i = 0; i < sizeof(original); i++) original[i] = i * 137U;
    FILEDESCRIPTORW files[2] = {0};
    files[0].dwFlags = files[1].dwFlags = FD_FILESIZE | FD_ATTRIBUTES;
    files[0].cFileName[0] = 0x6587;
    files[0].nFileSizeLow = sizeof(original);
    files[1].cFileName[0] = 'z'; /* zero-byte file */
    BYTE* packed = NULL;
    UINT32 length = 0;
    assert(!cliprdr_serialize_file_list(files, 2, &packed, &length) && length == sizeof(descriptors));
    memcpy(descriptors, packed, length);
    free(packed);
    CLIPRDR_FORMAT format = { .formatId = 0xc013, .formatName = "FileGroupDescriptorW" };
    CLIPRDR_FORMAT_LIST list = { .numFormats = 1, .formats = &format };
    assert(!guac_rdp_cliprdr_format_list(&a.ctx, &list) && !strcmp(message, "pending"));
    CLIPRDR_FORMAT_DATA_RESPONSE data = {
        .common = { .msgFlags = CB_RESPONSE_OK, .dataLen = length }, .requestedFormatData = descriptors
    };
    assert(!guac_rdp_cliprdr_format_data_response(&a.ctx, &data));
    for (UINT32 pos = 0; pos < sizeof(original);) {
        assert(next_request.listIndex == 0 && next_request.nPositionLow == pos);
        UINT32 n = next_request.cbRequested;
        assert(n > 0 && n <= sizeof(original) - pos);
        CLIPRDR_FILE_CONTENTS_RESPONSE block = {
            .common.msgFlags = CB_RESPONSE_OK, .streamId = next_request.streamId,
            .requestedData = original + pos, .cbRequested = n
        };
        assert(!file_received(&a.ctx, &block));
        pos += n;
    }
    assert(strlen(message) == 32);
    snprintf(token, sizeof(token), "%s/%.32s", sid, message);
    b.settings.file_clipboard_path = other;
    file_install(b.rdp.clipboard, token);
    assert(!strcmp(message, "rejected"));
    b.settings.file_clipboard_path = dst;
    file_install(b.rdp.clipboard, token);
    CLIPRDR_FORMAT_LIST_RESPONSE ack = { .common.msgFlags = CB_RESPONSE_OK };
    file_list_ack(&b.ctx, &ack);
    assert(!strncmp(message, "installed:", 10) && !strcmp(message + 10, token));
    CLIPRDR_FORMAT_DATA_REQUEST get = { .requestedFormatId = FILE_FORMAT };
    assert(!guac_rdp_cliprdr_format_data_request(&b.ctx, &get));
    source_clear(a.rdp.clipboard->files); /* Already-open target survives unlink. */
    CLIPRDR_FILE_CONTENTS_REQUEST req = { .streamId = 728, .dwFlags = FILECONTENTS_RANGE };
    for (UINT32 pos = 0; pos < sizeof(original); pos += expected_length) {
        req.nPositionLow = pos;
        req.cbRequested = 100000;
        expected_length = sizeof(original) - pos;
        if (expected_length > req.cbRequested) expected_length = req.cbRequested;
        expected = original + pos;
        assert(!file_requested(&b.ctx, &req));
    }
    req.listIndex = 1; req.dwFlags = FILECONTENTS_SIZE;
    BYTE zero[8] = {0};
    expected = zero; expected_length = 8;
    assert(!file_requested(&b.ctx, &req));
    req.listIndex = 2; expected = NULL; expected_length = 0;
    assert(!file_requested(&b.ctx, &req));
    guac_rdp_clipboard_free(a.rdp.clipboard);
    guac_rdp_clipboard_free(b.rdp.clipboard);
    assert(!rmdir(src) && !rmdir(dst) && !rmdir(root));
    puts("CLIPRDR copy/paste verified: unicode, multi-file, empty file, multi-range bytes, expired source and owner isolation.");
}
