#!/usr/bin/env python3
"""Guacamole 1.6 / FreeRDP 2.11: enforce the backend-authorized session leaf pin.
No shared known_hosts or CA fallback can override a mismatching session pin.
"""
from pathlib import Path
p=Path('src/protocols/rdp/rdp.c')
s=p.read_text()
marker='#ifdef HAVE_FREERDP_VERIFYCERTIFICATEEX\n/**'
assert s.count(marker)==1
callback=r'''
#include <openssl/pem.h>
#include <openssl/evp.h>
#include <openssl/crypto.h>
#include <strings.h>

/* remote-desktop-fnos: each connection receives one backend-authorized leaf
 * fingerprint; accept only for this session, never save to a shared store. */
static int rdp_verify_session_certificate(freerdp* instance,
        const BYTE* data, size_t length, const char* hostname, UINT16 port,
        DWORD flags) {
    (void) flags;
    rdpContext* context = GUAC_RDP_CONTEXT(instance);
    guac_client* client = ((rdp_freerdp_context*) context)->client;
    guac_rdp_client* rdp_client = (guac_rdp_client*) client->data;
    guac_rdp_settings* settings = rdp_client->settings;
    const char* expected = settings->certificate_fingerprints;
    if (!expected || strlen(expected) != 71 || strncmp(expected, "sha256:", 7)
            || !hostname || strcasecmp(hostname, settings->hostname)
            || port != settings->port || !data || length > 262144) {
        guac_client_log(client, GUAC_LOG_WARNING, "Session certificate authorization missing");
        return 0;
    }
    BIO* bio = BIO_new_mem_buf(data, (int)length);
    X509* cert = bio ? PEM_read_bio_X509(bio, NULL, NULL, NULL) : NULL;
    unsigned char digest[EVP_MAX_MD_SIZE];
    unsigned int count = 0;
    char actual[65];
    int accepted = 0;
    if (cert && X509_cmp_current_time(X509_get0_notBefore(cert)) < 0
            && X509_cmp_current_time(X509_get0_notAfter(cert)) > 0
            && X509_digest(cert, EVP_sha256(), digest, &count) && count == 32) {
        for (unsigned int i = 0; i < 32; i++)
            snprintf(actual + i*2, 3, "%02x", digest[i]);
        accepted = CRYPTO_memcmp(actual, expected + 7, 64) == 0;
    }
    X509_free(cert);
    BIO_free(bio);
    if (!accepted) {
        guac_client_log(client, GUAC_LOG_WARNING, "Session certificate fingerprint mismatch or expired");
        return 0;
    }
    return 2;
}

'''
s=s.replace(marker,callback+marker)
marker='    guac_rdp_push_settings(client, settings, instance);'
assert s.count(marker)==1
s=s.replace(marker,marker+r'''

    /* Validate even a CA-trusted certificate against the authorized leaf.
     * Clear FreeRDP's additive allowlist so it cannot bypass this callback. */
    if (!freerdp_settings_set_bool(context->settings, FreeRDP_ExternalCertificateManagement, TRUE)
            || !freerdp_settings_set_bool(context->settings, FreeRDP_IgnoreCertificate, FALSE)
            || !freerdp_settings_set_bool(context->settings, FreeRDP_AutoAcceptCertificate, FALSE)
            || !freerdp_settings_set_string(context->settings, FreeRDP_CertificateAcceptedFingerprints, ""))
        return FALSE;
''')
marker='    rdp_inst->Authenticate = rdp_freerdp_authenticate;'
assert s.count(marker)==1
s=s.replace(marker,marker+'\n    rdp_inst->VerifyX509Certificate = rdp_verify_session_certificate;')
p.write_text(s)
