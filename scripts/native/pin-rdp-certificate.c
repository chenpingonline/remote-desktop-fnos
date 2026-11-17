/* Explicit, host-and-port-scoped certificate trust using FreeRDP's public API.
 * Reads only a public DER certificate from stdin. Never reads login credentials. */
#include <ctype.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <sys/stat.h>
#include <unistd.h>
#include <openssl/x509.h>
#include <openssl/evp.h>
#include <freerdp/crypto/certificate.h>
#include <freerdp/settings.h>
#include <winpr/ssl.h>

int main(int argc, char** argv) {
    if (argc != 4 || strlen(argv[3]) != 64) {
        fputs("Expected host, port, confirmed SHA256; public DER certificate on stdin\n", stderr); return 2;
    }
    for (const char* p = argv[1]; *p; p++) {
        if (isspace((unsigned char)*p) || iscntrl((unsigned char)*p)) return 2;
    }
    char* end = NULL;
    long port = strtol(argv[2], &end, 10);
    if (!*argv[1] || *end || port < 1 || port > 65535) return 2;
    for (int i=0;i<64;i++) if (!isxdigit((unsigned char)argv[3][i])) return 2;
    const char* home = getenv("HOME");
    const char* config = getenv("XDG_CONFIG_HOME");
    struct stat st;
    if (!home || !config || lstat(home,&st) || !S_ISDIR(st.st_mode) || st.st_uid!=geteuid() ||
        lstat(config,&st) || !S_ISDIR(st.st_mode) || st.st_uid!=geteuid()) {
        fputs("Run as the application account with its private runtime directories\n",stderr); return 3;
    }
    BYTE buffer[65537]; size_t length=fread(buffer,1,sizeof(buffer),stdin);
    if (!length || length>65536 || ferror(stdin)) return 4;
    const unsigned char* p=buffer;
    X509* cert=d2i_X509(NULL,&p,(long)length);
    if (!cert || p!=buffer+length) {X509_free(cert);return 4;}
    unsigned char digest[EVP_MAX_MD_SIZE];unsigned int count=0;char actual[65];
    if (!X509_digest(cert,EVP_sha256(),digest,&count) || count!=32) {X509_free(cert);return 4;}
    for (unsigned int i=0;i<32;i++) snprintf(actual+2*i,3,"%02x",digest[i]);
    if (strcasecmp(actual,argv[3])) {
        fputs("Certificate differs from the confirmed SHA256; trust was not changed\n",stderr);X509_free(cert);return 5;
    }
    umask(077);
    if (!winpr_InitializeSSL(WINPR_SSL_INIT_DEFAULT)) {X509_free(cert);return 6;}
    rdpSettings* settings=freerdp_settings_new(FALSE);
    rdpCertificateStore* store=settings?certificate_store_new(settings):NULL;
    char* fingerprint=crypto_cert_fingerprint(cert);
    char* subject=crypto_cert_subject(cert);
    char* issuer=crypto_cert_issuer(cert);
    rdpCertificateData* entry=(fingerprint&&subject&&issuer)?certificate_data_new(argv[1],(UINT16)port,subject,issuer,fingerprint):NULL;
    int result=6;
    if (store&&entry) {
        int match=certificate_data_match(store,entry);
        if (match==0) {puts("This exact certificate is already trusted for this host and port");result=0;}
        else if (match==-1) {fputs("A different certificate is already pinned; refusing to replace it\n",stderr);result=7;}
        else if (match==1 && certificate_data_print(store,entry)) {
            puts("Confirmed SHA256 pinned for this host and port; other certificates still require verification");result=0;
        }
    }
    if (entry) certificate_data_free(entry);
    if (store) certificate_store_free(store);
    if (settings) freerdp_settings_free(settings);
    free(fingerprint);free(subject);free(issuer);X509_free(cert);
    return result;
}
