/* Tests WinPR's actual NLA digest path with a public MD4 known-answer vector. */
#include <stdio.h>
#include <string.h>
#include <winpr/ssl.h>
#include <winpr/crypto.h>
int main(void) {
 const BYTE expected[16] = {0xa4,0x48,0x01,0x7a,0xaf,0x21,0xd8,0x52,0x5f,0xc1,0x0a,0xe8,0x7a,0xa6,0x72,0x9d};
 BYTE digest[16] = {0};
 if (!winpr_InitializeSSL(WINPR_SSL_INIT_DEFAULT) || !winpr_Digest(WINPR_MD_MD4, (const BYTE*)"abc", 3, digest, sizeof(digest)) || memcmp(digest, expected, sizeof(digest))) {
  fputs("NLA crypto self-test FAILED\n", stderr); return 1;
 }
 puts("NLA crypto self-test PASS"); return 0;
}
