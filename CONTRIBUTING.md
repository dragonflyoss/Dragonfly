# Contributing to Dragonfly

It is warmly welcomed if you have interest to hack on Dragonfly. First, we encourage this kind of willing very much. And here is a list of contributing guide for you.

## Topics

* [Reporting security issues](#reporting-security-issues)
* [Reporting general issues](#reporting-general-issues)
* [Code and doc contribution](#code-and-doc-contribution)
* [Engage to help anything](#engage-to-help-anything)

## Reporting security issues

Security issues are always treated seriously. As our usual principle, we discourage anyone to spread security issues. If you find a security issue of Dragonfly, please do not discuss it in public and even do not open a public issue. Instead we encourage you to send us a private email to  [dragonfly-dev@service.alibaba.com](mailto:dragonfly-dev@service.alibaba.com) to report this.

## Reporting general issues

To be honest, we regard every user of Dragonfly as a very kind contributor. After experiencing Dragonfly, you may have some feedback for the project. Then feel free to open an issue via [NEW ISSUE](https://github.com/dragonflyoss/Dragonfly/issues/new).

Since we collaborate project Dragonfly in a distributed way, we appreciate **WELL-WRITTEN**, **DETAILED**, **EXPLICIT** issue reports. To make the communication more efficient, we wish everyone could search if your issue is an existing one in the searching list. If you find it existing, please add your details in comments under the existing issue instead of opening a brand new one.

To make the issue details as standard as possible, we setup an [ISSUE TEMPLATE](./.github/ISSUE_TEMPLATE) for issue reporters. You can find three kinds of issue templates there: question, bug report and feature request. Please **BE SURE** to follow the instructions to fill fields in template.

There are lot of cases when you could open an issue:

* bug report
* feature request
* performance issues
* feature proposal
* feature design
* help wanted
* doc incomplete
* test improvement
* any questions on project
* and so on

Also we must remind that when filing a new issue, please remember to remove the sensitive data from your post. Sensitive data could be password, secret key, network locations, private business data and so on.

## Code and doc contribution

Every action to make project Dragonfly better is encouraged. On GitHub, every improvement for Dragonfly could be via a PR (short for pull request).

* If you find a typo, try to fix it!
* If you find a bug, try to fix it!
* If you find some redundant codes, try to remove them!
* If you find some test cases missing, try to add them!
* If you could enhance a feature, please **DO NOT** hesitate!
* If you find code implicit, try to add comments to make it clear!
* If you find code ugly, try to refactor that!
* If you can help to improve documents, it could not be better!
* If you find document incorrect, just do it and fix that!
* ...

Actually it is impossible to list them completely. Just remember one princinple:

> WE ARE LOOKING FORWARD TO ANY PR FROM YOU.

Since you are ready to improve Dragonfly with a PR, we suggest you could take a look at the PR rules here.

* [Workspace Preparation](#workspace-preparation)
* [Branch Definition](#branch-definition)
* [Commit Rules](#commit-rules)
* [PR Description](#pr-description)

### Workspace Preparation

To put forward a PR, we assume you have registered a GitHub ID. Then you could finish the preparation in the following steps:

1. **FORK** Dragonfly to your repository. To make this work, you just need to click the button Fork in right-left of [dragonflyoss/Dragonfly](https://github.com/dragonflyoss/Dragonfly) main page. Then you will end up with your repository in `https://github.com/<your-username>/Dragonfly`, in which `your-username` is your GitHub username.
1. **CLONE** your own repository to develop locally. Use `git clone https://github.com/<your-username>/Dragonfly.git` to clone repository to your local machine. Then you can create new branches to finish the change you wish to make.

1. **Set Remote** upstream to be `https://github.com/dragonflyoss/Dragonfly.git` using the following two commands:

```
git remote add upstream https://github.com/dragonflyoss/Dragonfly.git
git remote set-url --push upstream no-pushing
```

With this remote setting, you can check your git remote configuration like this:

```
$ git remote -v
origin     https://github.com/<your-username>/Dragonfly.git (fetch)
origin     https://github.com/<your-username>/Dragonfly.git (push)
upstream   https://github.com/dragonflyoss/Dragonfly.git (fetch)
upstream   no-pushing (push)
```

Adding this, we can easily synchronize local branches with upstream branches.

### Branch Definition

Right now we assume every contribution via pull request is for [branch master](https://github.com/dragonflyoss/Dragonfly/tree/master) in Dragonfly. Before contributing, be aware of branch definition would help a lot.

As a contributor, keep in mind again that every contribution via pull request is for branch master. While in project Dragonfly, there are several other branches, we generally call them rc branches, release branches and backport branches.

Before officially releasing a version, we will checkout a rc(release candidate) branch. In this branch, we will test more than branch master.

When officially releasing a version, there will be a release branch before tagging. After tagging, we will delete the release branch.

When backporting some fixes to existing released version, we will checkout backport branches. After backporting, the backporting effects will be in PATCH number in MAJOR.MINOR.PATCH of [SemVer](http://semver.org/).

### Commit Rules

Actually in Dragonfly, we take two rules serious when committing:

* [Commit Message](#commit-message)
* [Commit Content](#commit-content)

#### Commit Message

Commit message could help reviewers better understand what is the purpose of submitted PR. It could help accelerate the code review procedure as well. We encourage contributors to use **EXPLICIT** commit message rather than ambiguous message. In general, we advocate the following commit message type:

* docs: xxxx. For example, "docs: add docs about storage installation".
* feature: xxxx.For example, "feature: make result show in sorted order".
* bugfix: xxxx. For example, "bugfix: fix panic when input nil parameter".
* style: xxxx. For example, "style: format the code style of Constants.java".
* refactor: xxxx. For example, "refactor: simplify to make codes more readable".
* test: xxx. For example, "test: add unit test case for func InsertIntoArray".
* chore: xxx. For example, "chore: integrate travis-ci". It's the type of mantainance change.
* other readable and explicit expression ways.

On the other side, we discourage contributors from committing message like the following ways:

* ~~fix bug~~
* ~~update~~
* ~~add doc~~

#### Commit Content

Commit content represents all content changes included in one commit. We had better include things in one single commit which could support reviewer's complete review without any other commits' help. In another word, contents in one single commit can pass the CI to avoid code mess. In brief, there are two minor rules for us to keep in mind:

* avoid very large change in a commit;
* complete and reviewable for each commit.

No matter commit message, or commit content, we do take more emphasis on code review.

### PR Description

PR is the only way to make change to Dragonfly project files. To help reviewers better get your purpose, PR description could not be too detailed. We encourage contributors to follow the [PR template](./.github/PULL_REQUEST_TEMPLATE.md) to finish the pull request.

## Engage to help anything

We choose GitHub as the primary place for Dragonfly to collaborate. So the latest updates of Dragonfly are always here. Although contributions via PR is an explicit way to help, we still call for any other ways.

* reply to other's issues if you could;
* help solve other user's problems;
* help review other's PR design;
* help review other's codes in PR;
* discuss about Dragonfly to make things clearer;
* advocate Dragonfly technology beyond GitHub;
* write blogs on Dragonfly and so on.

In a word, **ANY HELP IS CONTRIBUTION.**
