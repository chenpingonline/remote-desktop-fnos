#!/usr/bin/env python3
"""Avoid eager Kerberos credential acquisition for accounts without a domain.

Keep NLA, certificate validation and explicit domain-account negotiation intact.
FreeRDP 3 Negotiate eagerly acquires Kerberos credentials before trying NTLM;
email usernames can therefore block on realm discovery before NTLM succeeds.
"""
from pathlib import Path

p = Path('src/protocols/rdp/rdp.c')
s = p.read_text()
anchor = '    guac_rdp_push_settings(client, settings, instance);'
assert s.count(anchor) == 1
s = s.replace(anchor, anchor + r'''

    /* A blank domain is the app's non-domain/password-account mode. Preserve
     * automatic negotiation for explicit AD domains (including DOMAIN\user).
     * MicrosoftAccount is a local Windows identity provider, not an AD realm. */
    const char* auth_domain = settings->domain;
    const char* auth_user = settings->username;
    BOOL microsoft_account = (auth_domain && !strcasecmp(auth_domain, "MicrosoftAccount"))
        || (auth_user && !strncasecmp(auth_user, "MicrosoftAccount\\", 17));
    BOOL unqualified_account = (!auth_domain || !*auth_domain)
        && (!auth_user || !strchr(auth_user, '\\'));
    if (microsoft_account || unqualified_account) {
        if (!freerdp_settings_set_string(context->settings,
                FreeRDP_AuthenticationPackageList, "none,ntlm"))
            return FALSE;
    }
''')
p.write_text(s)
