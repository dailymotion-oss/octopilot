---
title: "Sops"
anchor: "sops"
weight: 40
---

The **sops** updater can manipulate files encrypted with [mozilla's sops](https://github.com/mozilla/sops) natively, without the need to install [sops](https://github.com/mozilla/sops). This is great if you want to store sensitive data in your git repositories: you can encrypt them with `sops`, and use Octopilot to update them automatically.

For example if you want to store your TLS certificate as a base64 encoded string in a sops-encrypted file:

```bash
$ octopilot \
    --update `sops(file=secrets.yaml,key=app.tls.base64encodedCertificateKey)=$(kubectl -n cert-manager get secrets tls-myapp -o template='{{index .data "tls.key"}}')` \
    ...
```

Given the following (decrypted) `secrets.yaml` file:

```yaml
app:
  tls:
    base64encodedCertificateKey: LS0tLS1CRUdJTiBSU0EgU...
```

Octopilot will decrypt the `secrets.yaml` file, set the value of the `app.tls.base64encodedCertificateKey` key to the given value, and re-encrypt the `secrets.yaml` file before writing it to disk.

The syntax is: `sops(params)=value` - you can read more about the value in the ["value" section](#value).

It supports the following parameters:

- `file` (string): mandatory path to the file to update. Can be a file pattern - such as `config/secrets.*` to match files in the same directory, or `config/**/secrets.*` using double asterisks (**) to match files in subdirectories. If it's a relative path, it will be relative to the root of the cloned git repository. For more information on using file patterns, you can refer to the [go-zglob documentation](https://github.com/mattn/go-zglob).
- `key` (string): mandatory key to update in the file(s).

Note that depending on the sops backend you use (KMS, age, vault, ...) you might need to set some environment variables, such as:
- for GCP KMS, the `GOOGLE_APPLICATION_CREDENTIALS` env var
- for [age](https://age-encryption.org/), the `SOPS_AGE_KEY_FILE` env var
- ...

See the ["updating certificates" use-case](#use-case-update-certs) for a real-life example of what you can do with this updater.
