## apisix-ingress-controller release process

1. Create release pull request with release notes.

   1. Compile release notes detailing features added since the last release and
      add release template file to `releases/` directory. The template is defined
      by containerd's release tool but refer to previous release files for style
      and format help. Name the file using the version.
      See [release-tool](https://github.com/containerd/release-tool)

      You can use the following command to generate content

      ```
      release-tool -l -d -n -t 1.0.0 releases/v1.0.0.toml
      ```

2. Vote for release

3. Create tag

4. Push tag and Github release

5. Promote on Slack, Twitter, mailing lists, etc
