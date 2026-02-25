# Using the SAS Open Source Project Starter Kit
Use the SAS Open Source Project Starter Kit to stage a GitHub repository for a new open source project that follows the [SAS Open Source Contributions](https://inside.sas.com/policies/open-source-contributions/) policy.
See the Open source Program Office's guide to creating [First-Party Open Source Contributions](https://rndconfluence.sas.com/x/8TvSFg) for additional information.

## How to Use This Kit

1. Navigate to `https://github.com/organizations/sas-institute-rnd-internal/repositories/new`.
1. Click the __Repository template__ drop-down menu.
1. Select `sas-institute-rnd-internal/workflows-sas-open-source-project-starter-kit` as your new project's template.
1. Enter a project name that begins with `tmp-`.
1. Enter a description for the project.
1. Select a visibility level for the project.
1. Click __Create repository__.

> [!NOTE]
> If you do not select the `Internal` visibility setting, you must specifically provide reviewers access to your repository at a later time.

GitHub uses the SAS Open Source Project Starter Kit to create a new repository for your work, pre-populating that repository with all of the kit's materials.

## About the Kit's Components
The SAS Open Source Project Starter Kit contains the following components.

### README.md
We want our open source projects to be useful, so include a README file to help people get started using yours.
State plainly what your project does.
Offer examples.
Additional documentation is strongly encouraged in a format of your choosing, but a README file is the minimum you should offer.
The README file included in the project starter kit is annotated with guidelines for using it.
Some sections of the template are required; others are optional.

### CONTRIBUTING.md and ContributorAgreement.txt
The open source project starter kit includes a basic CONTRIBUTING.md file with the minimal contribution guidelines needed for your project.
The file is annotated; review these annotations to determine how to format this file.
If your project will accept contributions and patches from external community contributors, then edit this file to say so.
Every project that accepts contributions must include this file, along with the standard ContributorAgreement.txt file, which is also included in this kit.
If your project will not accept contributions and patches from external community contributors, likewise edit this file to say so.

### SUPPORT.md
The open source project starter kit includes a standard SUPPORT.md file that directs users to use GitHub issues and pull requests for support.
Any alternate means of support for your project first needs approval from SAS Legal and Technical Support and should be handled as part of the open source approval process prior to publishing your project on GitHub.

### SECURITY.md
Use this file if you plan to activate GitHub's private security vulnerability reporting for public projects.
The file is annotated with instructions for using this feature.
Modify the file to reflect your project's security posture.

### LICENSE
Your project must contain a file named LICENSE in its top-level directory.
The file must contain a copy of the project's license.
The starter kit contains a copy of SAS's default open source license, the Apache License version 2.0.

## Creating Project Documentation
The SAS Open Source Project Starter Kit contains resources for building project documentation that complies with SAS brand standards.
This documentation is built and served with the [Docusaurus](https://docusaurus.io/) website generator.
If you would like to use these documentation materials, edit the `website/docusaurus.config.ts` file in order to replace `<projectName>` with your project name.

> [!NOTE]
> The `docusaurus.config.ts` file contains multiple instances of this variable; be sure to locate and change them all.

Add Markdown files to `website/docs` to begin creating project documentation.
The website is automatically rebuilt when changes to these files are merged to the project's `main` branch.
See its [README](./website/README.md) for details.

See project documentation for the [SAS extension for Visual Studio Code](https://github.com/sassoftware/vscode-sas-extension/tree/main/website) for an example.

## Preparing Your Project for Review
Members of the Open Source Program Office and additional subject matter experts will review your project prior to release, as per the [Open Source Contributions](https://inside.sas.com/policies/open-source-contributions/) policy.

To ensure timely review of your work, complete these preparation steps.

### Add source code headers
Every file containing source code must include copyright and license information.
Place required source code headers at the top of the source code files, above any other header information.
This is to ensure that automated code scanning and license management tools (increasingly popular at SAS and elsewhere) will properly locate the copyright notices.

> [!NOTE]
> Source code refers to any executable code such as .java or .go files, shell scripts, etc.
> This includes any JS/CSS files that you might be serving out to browsers.
> It does not refer to content such as documentation, build scripts, or configuration files.

The following header template contains the minimum requirements:

```
Copyright © 2025, SAS Institute Inc., Cary, NC, USA.  All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
```

### Remove unnecessary files

When you have completed your work, delete the following files and folders from your repository:

- `INSTRUCTIONS.md` file
- `.github/CODEOWNERS` file
- `.github/sas-settings` directory

Additionally, if you have not used the kit's documentation materials, remove the following components from the repository:

- the `website` directory and all files that it contains
- the `Update Documentation` section of the `CONTRIBUTING.md` file
- `.github/dependabot.yml`
- `.github/workflows/deploy-doc.yml`

### Review the Final Preparation Checklist
The Open Source Program Office uses the following checklist to ensure are first-party open source projects are ready for public release.
By reviewing the checklist and ensuring your project's alignment with, you help expedite approval of your Open Source Contributions request.

- [ ] Properly formatted README.md file is present
- [ ] Properly formatted SUPPORT.md file is present
- [ ] Properly formatted CONTRIBUTING.md file is present
- [ ] Properly formatted SECURITY.md file is present if necessary
- [ ] Copy of SAS Contributor Agreement is present if necessary
- [ ] LICENSE file matches approved license
- [ ] Source code files contain required headers
- [ ] INSTRUCTIONS.md file is removed
- [ ] Comment blocks in template files are removed
- [ ] CODEOWNERS file is removed from .github folder
- [ ] sas-settings folder is removed from .github folder
- [ ] website directory is removed if not in use
- [ ] dependabot.yml file is removed from .github folder if not in use
- [ ] deploy-doc.yml file is removed from .github/workflows if not in use
- [ ] Git history has been scrubbed if necessary
- [ ] Remaining guidance from Legal has been addressed

## Submit Your Project for Review
To submit your open source project for review, follow the [Open Source Contributions Quick Start Guide](https://rndconfluence.sas.com/x/sKjrFg).
You provide links to your project repository and its materials as part of this review.
