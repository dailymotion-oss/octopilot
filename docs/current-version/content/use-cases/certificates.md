---
title: "Updating certificates"
anchor: "use-case-update-certs"
weight: 20
---

One of the question you will have to answer when doing gitops is: "where do I stop?". What do you store in your environment git repository? For example, should you store your certificates there too, or do you consider them "outside" of the gitops-scope, and so managed by something else, such as [cert-manager](https://cert-manager.io/)?

Let's say we want to manage them with our gitops process. It means that we'll need to:
- store them in a git repository. It's not a problem for the certificate itself, but we'll need to ensure that the private key won't be stored in clear text.
- update the git repository every time a new certificate is issued - ideally by automatically creating a Pull Request.

In a Kubernetes environment where you have multiple clusters - for example in different regions of the world, different environments, etc - we can setup and use [cert-manager](https://cert-manager.io/) to manage the certificates from a "central" cluster. No need to setup cert-manager in all your clusters. And we'll use Octopilot to "propagate" the certificates too all the clusters, through a gitops workflow, by storing and updating the certificates in one or more git repositories.

You can setup a nightly CronJob, a scheduled pipeline, or anything else you prefer to regularly call Octopilot, to make sure that all the certificates stored in the git repositories are up-to-date - or to create Pull Requests to update them.

In fact, we won't call Octopilot directly, we'll call a script that will perform a few operations before executing Octopilot:
- first, we'll need to retrieve all the certificates from the Kubernetes API - using something like `kubectl -n cert-manager get certificates.cert-manager.io`
- then, for each certificate, we'll need to retrieve its associated Kubernetes Secret, which contains the actual certificate and its private key - both base64-encoded - using something like `kubectl -n cert-manager get secret $secretName -o go-template='{{index .data "tls.crt"}}' > tls.crt.base64`
- optionally, we can extract some data from the certificates, such as the DNS names, validity dates, and so on - to generate nice commit messages and Pull requests. Use the `openssl` tool to extract the `startdate` and `enddate` fields for example.
- and then we can execute Octopilot with something like the following:

```
$ octopilot \
    --repo "owner/prod-env" \
    --update "yaml(file=${APP_NAME}-values.yaml,path=tls.certificate)=file(path=tls.crt.base64)" \
    --update "sops(file=${APP_NAME}-secrets.yaml,key=tls.certificateKey)=file(path=tls.key.base64)" \
    --git-commit-title "Update ${APP_NAME} certificate" \
    --git-commit-body "..." \
    --git-branch-prefix "octopilot-cert-${APP_NAME}-" \
    --pr-labels "update-cert-${APP_NAME}"
```

Here, we are running 2 [updaters](#updaters):
- the [YAML updater](#yaml), to update the Helm values file of our application, and set the `tls.certificate` value to the content of a `tls.cert.base64` file
- the [SOPS updater](#sops), to update the Helm secrets file of our application - which is in fact a [sops](https://github.com/mozilla/sops)-encrypted YAML file - and set the `tls.certificateKey` value to the content of a `tls.key.base64` file

We're using the default `reset` strategy, which means that if we don't merge the PR right away, and your CronJob runs a second time, it will find the existing PR and just reset it from the base branch, and then force push the commit. So you'll always only see 1 commit, rebased every day, with the latest certificate. We're using a specific label for our Pull Request: `octopilot-cert-${APP_NAME}` - to make sure we'll find any existing PR for our application/certificate, and that each application/certificate will get its own PR.

## Result

This is a screenshot of a Pull Request on an environment git repository, which updates a specific certificate.

![](screenshot-cert-pr.png)

You can see that we extracted a few information, to make it easy for reviewers - because the diff is just a base64 blob replaced by another one:
- the creation date of the new certificate - in fact this is the creation timestamp of the latest cert-manager order, which means the latest time the certificate has been renewed
- the new validity dates, extracted using `openssl x509 -in tls.crt -noout -startdate` for example
- the validity dates of the currently deployed certificate, retrieved using `openssl s_client -servername $domain -connect $domain:443 > previous.crt` for example, and then extracted using `openssl` once again
- the list of domains - or DNS names - for which this certificate is valid, extracted using `kubectl -n cert-manager get $cert -o go-template={{.spec.dnsNames}}` for example

Merging this Pull Request would result in a re-deployment of our application, with an updated certificate. So with this approach we can control when we want to deploy new certificates.
