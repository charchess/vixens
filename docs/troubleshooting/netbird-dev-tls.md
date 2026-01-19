# Netbird TLS Issues in Dev Environment

## Problem
When connecting a Netbird client to the Dev environment (`netbird.dev.truxonline.com`), you may encounter the following error:
```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

## Cause
The Dev environment uses **Let's Encrypt Staging** certificates to avoid rate limits. These certificates are signed by a Staging Root CA (`(STAGING) Pretend Pear X1` or `Fake LE Root X1`) which is not trusted by default in most operating systems.

## Solution

### For Linux/macOS
You need to add the Let's Encrypt Staging Root CA to your system's trust store.

1. **Download the Staging Root CA:**
   ```bash
   curl -o letsencrypt-stg-root-x1.pem https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem
   ```

2. **Trust the Certificate:**
   - **Debian/Ubuntu:**
     ```bash
     sudo cp letsencrypt-stg-root-x1.pem /usr/local/share/ca-certificates/
     sudo update-ca-certificates
     ```
   - **Fedora/CentOS:**
     ```bash
     sudo cp letsencrypt-stg-root-x1.pem /etc/pki/ca-trust/source/anchors/
     sudo update-ca-trust
     ```
   - **macOS:**
     ```bash
     sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain letsencrypt-stg-root-x1.pem
     ```

### For Windows
If you use the Netbird Windows Client, you must manually install the Staging Root CA into the Windows Trust Store.

1. **Download the Staging Root CA:** [letsencrypt-stg-root-x1.pem](https://letsencrypt.org/certs/staging/letsencrypt-stg-root-x1.pem)
2. **Install the Certificate:**
   - Right-click the downloaded file -> **Install Certificate**.
   - Select **Local Machine** (requires admin rights).
   - Select **Place all certificates in the following store**.
   - Click **Browse** and select **Trusted Root Certification Authorities**.
   - Complete the wizard.
3. **Restart Netbird:** Close and reopen your terminal or restart the Netbird service.

### For Netbird Service (in Cluster)
The `netbird-management` service in Dev has already been patched to trust this CA via the `netbird-ca-bundle` ConfigMap.
