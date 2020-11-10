# SOPS Test Data

This directory contains everything required to run the unit-tests for the [sops](https://github.com/mozilla/sops) integration.

For the unit-tests we use a PGP master key, which has been generated using the following command:

```
$ GNUPGHOME=.gnupg gpg --batch --gen-key gpg-gen-key-data.txt
```

It requires the [gpg](https://gnupg.org/) tool, and uses the data from the [gpg-gen-key-data.txt](gpg-gen-key-data.txt) file to create a key. This command should be run from inside this directory (`testdata`), and will write files inside the `.gnupg` directory.
Note that the key doesn't have any passphrase, to make it easier to use in a unit-testing environment.

If you have an issue with the gpg command, with an error message such as: `gpg: can't connect to the agent: IPC connect call failed`, it may be because the `GNUPGHOME` path is too long. You can confirm this by running `GNUPGHOME=.gnupg gpg-agent --daemon -v`, and if it fails with "socket name is too long", then you'll need to move this repo up in your FS hierarchy.

The key fingerprint was then retrieved using the following command:

```
$ GNUPGHOME=.gnupg gpg --list-secret-keys --keyid-format LONG
```

In an output such as

```
sec   rsa2048/F7D394865A2FE709 2020-01-13 [SCEA]
      4C671A61C17E5399FBE235ACF7D394865A2FE709
uid                [  ultime ] Octo Pilot <octopilot@example.com>
ssb   rsa2048/9452B1F4319DFCF1 2020-01-13 [SEA]
```

The fingerprint we care about is `F7D394865A2FE709` - this is the part we need to put in the [sops_test.go](sops_test.go) file to identify the master key to use to encrypt/decrypt files using sops.

All those operations have already been done once, and you should not need to do anything unless you want to regenerate a new key for some reason.
