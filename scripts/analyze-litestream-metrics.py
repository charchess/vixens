#!/usr/bin/env python3
"""
Analyze Litestream backup metrics from MinIO bucket.

Usage:
    python3 scripts/analyze-litestream-metrics.py

Requirements:
    pip install boto3 tabulate
    
Environment variables:
    LITESTREAM_ENDPOINT (default: http://192.168.111.69:9001)
    LITESTREAM_BUCKET (default: vixens-litestream)
    AWS_ACCESS_KEY_ID (from Infisical)
    AWS_SECRET_ACCESS_KEY (from Infisical)
"""

import boto3
from datetime import datetime, timedelta
from collections import defaultdict
from tabulate import tabulate
import os
import re

# MinIO/S3 Configuration
ENDPOINT = os.getenv('LITESTREAM_ENDPOINT', 'http://192.168.111.69:9001')
BUCKET = os.getenv('LITESTREAM_BUCKET', 'vixens-litestream')
ACCESS_KEY = os.getenv('AWS_ACCESS_KEY_ID')
SECRET_KEY = os.getenv('AWS_SECRET_ACCESS_KEY')

def analyze_litestream_bucket():
    """Analyze Litestream backup structure in MinIO."""
    
    # Connect to MinIO
    s3 = boto3.client(
        's3',
        endpoint_url=ENDPOINT,
        aws_access_key_id=ACCESS_KEY,
        aws_secret_access_key=SECRET_KEY
    )
    
    # List all objects
    print(f"🔍 Analyzing bucket: {BUCKET}")
    print(f"📡 Endpoint: {ENDPOINT}\n")
    
    paginator = s3.get_paginator('list_objects_v2')
    pages = paginator.paginate(Bucket=BUCKET)
    
    # Statistics per app/database
    stats = defaultdict(lambda: {
        'snapshots': 0,
        'wal_files': 0,
        'total_size': 0,
        'oldest': None,
        'newest': None,
        'snapshot_sizes': [],
        'wal_sizes': []
    })
    
    for page in pages:
        if 'Contents' not in page:
            continue
            
        for obj in page['Contents']:
            key = obj['Key']
            size = obj['Size']
            modified = obj['LastModified']
            
            # Parse key structure: app-name/db-name/generation/type/file
            # Example: hydrus-client/client.db/abc123/snapshots/00000001.snapshot
            parts = key.split('/')
            if len(parts) < 3:
                continue
                
            app_db = '/'.join(parts[:2])  # e.g., "hydrus-client/client.db"
            
            # Detect file type
            is_snapshot = '.snapshot' in key or '/snapshots/' in key
            is_wal = '.wal' in key or '/wal/' in key
            
            # Update stats
            s = stats[app_db]
            s['total_size'] += size
            
            if is_snapshot:
                s['snapshots'] += 1
                s['snapshot_sizes'].append(size)
            elif is_wal:
                s['wal_files'] += 1
                s['wal_sizes'].append(size)
            
            # Track age
            if s['oldest'] is None or modified < s['oldest']:
                s['oldest'] = modified
            if s['newest'] is None or modified > s['newest']:
                s['newest'] = modified
    
    # Display results
    print("=" * 100)
    print("📊 LITESTREAM BACKUP METRICS")
    print("=" * 100)
    print()
    
    # Summary table
    table_data = []
    now = datetime.now(stats[list(stats.keys())[0]]['newest'].tzinfo) if stats else datetime.now()
    
    for app_db, s in sorted(stats.items()):
        age_days = (now - s['oldest']).days if s['oldest'] else 0
        retention_days = (now - s['oldest']).days if s['oldest'] else 0
        
        avg_snapshot = sum(s['snapshot_sizes']) / len(s['snapshot_sizes']) if s['snapshot_sizes'] else 0
        avg_wal = sum(s['wal_sizes']) / len(s['wal_sizes']) if s['wal_sizes'] else 0
        
        # Estimate writes per day (WAL files / age)
        wal_per_day = s['wal_files'] / max(age_days, 1) if age_days > 0 else s['wal_files']
        
        table_data.append([
            app_db,
            s['snapshots'],
            s['wal_files'],
            f"{wal_per_day:.1f}",
            f"{s['total_size'] / 1024 / 1024:.1f} MB",
            f"{avg_snapshot / 1024 / 1024:.1f} MB" if avg_snapshot else "N/A",
            f"{retention_days}d"
        ])
    
    headers = ["App/Database", "Snapshots", "WAL Files", "WAL/day", "Total Size", "Avg Snapshot", "Retention"]
    print(tabulate(table_data, headers=headers, tablefmt="grid"))
    print()
    
    # Profile recommendations
    print("=" * 100)
    print("🎯 PROFILE RECOMMENDATIONS")
    print("=" * 100)
    print()
    
    for app_db, s in sorted(stats.items()):
        age_days = (now - s['oldest']).days if s['oldest'] else 1
        wal_per_day = s['wal_files'] / max(age_days, 1)
        
        # Heuristic: WAL/day as activity indicator
        if wal_per_day > 100:
            profile = "CRITICAL (1h snapshots)"
        elif wal_per_day > 20:
            profile = "STANDARD (6h snapshots)"
        else:
            profile = "RELAXED (24h snapshots)"
        
        print(f"📦 {app_db:40s} → {profile:30s} ({wal_per_day:.1f} WAL/day)")
    
    print()
    
    # Cleanup recommendations
    print("=" * 100)
    print("🧹 CLEANUP ANALYSIS")
    print("=" * 100)
    print()
    
    total_size_gb = sum(s['total_size'] for s in stats.values()) / 1024 / 1024 / 1024
    print(f"Total storage used: {total_size_gb:.2f} GB")
    print()
    
    # Find old files (>7 days)
    old_threshold = now - timedelta(days=7)
    for app_db, s in stats.items():
        if s['oldest'] and s['oldest'] < old_threshold:
            old_days = (now - s['oldest']).days
            if old_days > 7:
                print(f"⚠️  {app_db}: Has files {old_days} days old (cleanup recommended)")

if __name__ == '__main__':
    if not ACCESS_KEY or not SECRET_KEY:
        print("❌ Error: AWS_ACCESS_KEY_ID and AWS_SECRET_ACCESS_KEY must be set")
        print("   Get from: kubectl get secret -n argocd infisical-universal-auth -o jsonpath='{.data}'")
        exit(1)
    
    try:
        analyze_litestream_bucket()
    except Exception as e:
        print(f"❌ Error: {e}")
        exit(1)
