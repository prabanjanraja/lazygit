# GitHub Workflows Documentation

This document describes the GitHub Actions workflows used in this repository for managing releases and brew tap updates.

## Overview

There are three main workflows that work together to maintain synchronization with the upstream repository and manage the custom brew tap:

1. Watch Upstream Changes (`watch-upstream.yml`)
2. Sync Upstream Release (`sync-upstream-release.yml`)
3. Update Brew Tap (`update-brew-tap.yml`)

## Workflow Details

### 1. Watch Upstream Changes (`watch-upstream.yml`)

**Purpose**: Monitors the original lazygit repository and Homebrew formula for updates.

**Triggers**:
- Weekly schedule (Monday at 00:00)
- Manual trigger (workflow_dispatch)

**Actions**:
- Checks for new releases in jesseduffield/lazygit
- Checks for updates in the Homebrew core formula
- Triggers the Sync Upstream Release workflow when changes are detected

### 2. Sync Upstream Release (`sync-upstream-release.yml`)

**Purpose**: Creates a new release in your fork when there's a new upstream release.

**Triggers**:
- Repository dispatch events (upstream_release, formula_update)
- Manual trigger (workflow_dispatch)
- Watch event

**Actions**:
- Fetches latest release information from upstream
- Creates matching tag and release in your fork
- Copies release notes and version information

### 3. Update Brew Tap (`update-brew-tap.yml`)

**Purpose**: Updates your custom Homebrew tap with the latest release.

**Triggers**:
- When a release is published in your fork
- Manual trigger (workflow_dispatch)

**Actions**:
- Gets release information from your fork
- Calculates SHA256 for the release tarball
- Updates the formula in your homebrew-tap repository
- Commits and pushes the changes

## Workflow Interaction

The workflows form a chain of automation:

1. `watch-upstream.yml` detects changes in the original repository
2. This triggers `sync-upstream-release.yml` to create a new release in your fork
3. The new release triggers `update-brew-tap.yml` to update your custom formula

## Setup Requirements

1. A GitHub repository named `homebrew-tap` in your account
2. A Personal Access Token (PAT) with repository access stored as `TAP_REPO_TOKEN`

## Manual Triggers

All workflows can be triggered manually from the GitHub Actions interface if needed.

## Installation for Users

Users can install your version using:
```bash
brew install <your-github-username>/tap/lazygit
``` 