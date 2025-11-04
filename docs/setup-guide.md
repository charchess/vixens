# Setup Guide

## 1. Introduction

This document provides instructions for setting up the Vixens project.

## 2. Prerequisites

The following prerequisites are required to run this project:

- [Terraform](https://www.terraform.io/downloads.html)
- [Talos](https://www.talos.dev/v1.6/introduction/getting-started/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl-linux/)
- [talosctl](https://www.talos.dev/v1.6/introduction/getting-started/#talosctl)

## 3. Installation

1. Clone the repository:

   ```bash
   git clone https://github.com/charchess/vixens.git
   ```

2. Navigate to the project directory:

   ```bash
   cd vixens
   ```

## 4. Configuration

1. Navigate to the `terraform/environments/dev` directory:

   ```bash
   cd terraform/environments/dev
   ```

2. Create a `terraform.tfvars` file and populate it with the required variables.

## 5. Usage

1. Initialize Terraform (from project root):

   ```bash
   terraform -chdir=terraform/environments/dev init -upgrade
   ```

2. Apply the Terraform configuration (from project root):

   ```bash
   terraform -chdir=terraform/environments/dev apply -auto-approve
   ```
