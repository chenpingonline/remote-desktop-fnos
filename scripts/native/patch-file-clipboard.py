"""Wire the file clipboard into the pinned Guacamole 1.6.0 RDP adapter."""
from pathlib import Path

root = Path("src/protocols/rdp")
def replace(file, old, new):
    path = root / file
    src = path.read_text()
    assert src.count(old) == 1, (file, old)
    path.write_text(src.replace(old, new))

replace("settings.c", '    "disable-copy",', '    "file-clipboard-path",\n    "disable-copy",')
replace("settings.c", '    IDX_DISABLE_COPY,\n', '    IDX_FILE_CLIPBOARD_PATH,\n    IDX_DISABLE_COPY,\n')
replace("settings.c", "    /* Parse clipboard copy disable flag */", '''
    settings->file_clipboard_path = guac_user_parse_args_string(user, GUAC_RDP_CLIENT_ARGS,
            argv, IDX_FILE_CLIPBOARD_PATH, NULL);
    /* Parse clipboard copy disable flag */''')
replace("settings.c", "    guac_mem_free(settings->drive_path);", "    guac_mem_free(settings->file_clipboard_path);\n    guac_mem_free(settings->drive_path);")
replace("settings.h", "    int disable_copy;", "    int disable_copy;\n    char* file_clipboard_path;")
replace("rdp.c", "if (!(settings->disable_copy && settings->disable_paste))", "if (settings->file_clipboard_path || !(settings->disable_copy && settings->disable_paste))")
replace("user.c", "if (!settings->disable_paste)", "if (settings->file_clipboard_path || !settings->disable_paste)")
replace("channels/cliprdr.h", "    UINT requested_format;", "    UINT requested_format;\n    struct file_clipboard* files;")
c = "channels/cliprdr.c"
replace(c, "#include <assert.h>", '#include "file-clipboard.c"\n\n#include <assert.h>')
replace(c, ".generalFlags = CB_USE_LONG_FORMAT_NAMES", ".generalFlags = CB_USE_LONG_FORMAT_NAMES | (file_directory(clipboard) ? CB_STREAM_FILECLIP_ENABLED | CB_FILECLIP_NO_FILE_PATHS : 0)")
replace(c, "    /* Prefer Unicode (in this case, UTF-16) */", '''
    if (file_directory(clipboard)) {
        file_clipboard* f = clipboard->files;
        target_clear(f, 0);
        int had_files = f->source_path || f->file_format;
        source_clear(f);
        f->file_format = 0;
        for (UINT32 i = 0; i < format_list->numFormats; i++) {
            const CLIPRDR_FORMAT* format = &format_list->formats[i];
            if (format->formatName && !strcmp(format->formatName, "FileGroupDescriptorW")
                    && format->formatId >= 0xc000) {
                f->file_format = format->formatId;
                break;
            }
        }
        UINT32 wanted = f->file_format;
        if (wanted) file_event(clipboard, "pending");
        else {
            if (had_files) file_event(clipboard, "clear");
            if (!rdp_client->settings->disable_copy) {
                if (guac_rdp_cliprdr_format_supported(format_list, CF_UNICODETEXT)) wanted = CF_UNICODETEXT;
                else if (guac_rdp_cliprdr_format_supported(format_list, CF_TEXT)) wanted = CF_TEXT;
            }
        }
        /* FormatDataResponse has no request ID. Drain any outstanding request
         * before issuing one for the replacement clipboard generation. */
        if (f->format_pending) {
            f->superseded = 1;
            f->queued_format = wanted;
            return CHANNEL_RC_OK;
        }
        if (!wanted) return CHANNEL_RC_OK;
        f->format_pending = 1;
        return guac_rdp_cliprdr_send_format_data_request(cliprdr, wanted);
    }

    /* Prefer Unicode (in this case, UTF-16) */''')
replace(c, "    guac_iconv_write* remote_writer;", '''
    if (file_directory(clipboard) && format_data_request->requestedFormatId == FILE_FORMAT)
        return file_descriptors_requested(cliprdr);
    if (settings->disable_paste) {
        CLIPRDR_FORMAT_DATA_RESPONSE failed = { .common.msgFlags = CB_RESPONSE_FAIL };
        return cliprdr->ClientFormatDataResponse(cliprdr, &failed);
    }
    guac_iconv_write* remote_writer;''')
replace(c, "    /* Ignore received data if copy has been disabled */", '''
    if (file_directory(clipboard)) {
        file_clipboard* f = clipboard->files;
        if (!f->format_pending) return CHANNEL_RC_OK;
        f->format_pending = 0;
        if (f->superseded) {
            f->superseded = 0;
            if (!f->queued_format) return CHANNEL_RC_OK;
            f->format_pending = 1;
            return guac_rdp_cliprdr_send_format_data_request(cliprdr, f->queued_format);
        }
        if (f->file_format && clipboard->requested_format == f->file_format)
            return source_descriptors(cliprdr, format_data_response);
    }
    /* Ignore received data if copy has been disabled */''')
replace(c, "    cliprdr->MonitorReady =", '''    if (file_directory(clipboard)) {
        cliprdr->ServerFileContentsRequest = file_requested;
        cliprdr->ServerFileContentsResponse = file_received;
        cliprdr->ServerFormatListResponse = file_list_ack;
    }
    cliprdr->MonitorReady =''')
replace(c, "    clipboard->client = client;", '''    clipboard->files = guac_mem_zalloc(sizeof(file_clipboard));
    pthread_mutex_init(&clipboard->files->target_lock, NULL);
    clipboard->client = client;''')
replace(c, "    guac_common_clipboard_free(clipboard->clipboard);", '''    source_clear(clipboard->files);
    snapshot_close(&clipboard->files->target);
    pthread_mutex_destroy(&clipboard->files->target_lock);
    guac_mem_free(clipboard->files);
    guac_common_clipboard_free(clipboard->clipboard);''')
replace(c, "    /* Handle any future \"blob\" and \"end\" instructions", '''    if (!strcmp(mimetype, FILE_MIME)) {
        if (file_directory(clipboard)) stream->blob_handler = file_token_blob;
        return 0;
    }
    if (rdp_client->settings->disable_paste || strcmp(mimetype, "text/plain")) return 0;

    /* Handle any future "blob" and "end" instructions''')

replace(c, "    int retval = cliprdr->ClientFormatList(", "    if (file_directory(clipboard)) target_clear(clipboard->files, 1);\n    int retval = cliprdr->ClientFormatList(")
