#!/usr/bin/env python3
"""
Superset Multi-Tenant Security Configuration

This script configures Apache Superset with tenant-specific roles and RLS filters
to enforce multi-tenant data isolation at the analytics layer.
"""

import requests
import json
import os
import sys
from typing import Dict, List, Optional

# Superset configuration
SUPERSET_URL = os.getenv("SUPERSET_URL", "http://localhost:8088")
SUPERSET_USERNAME = os.getenv("SUPERSET_USERNAME", "admin")
SUPERSET_PASSWORD = os.getenv("SUPERSET_PASSWORD", "admin")

class SupersetSecurityManager:
    def __init__(self, base_url: str, username: str, password: str):
        self.base_url = base_url.rstrip('/')
        self.session = requests.Session()
        self.access_token = None
        self.refresh_token = None
        
    def login(self) -> bool:
        """Authenticate with Superset and get access token"""
        url = f"{self.base_url}/api/v1/security/login"
        payload = {
            "username": username,
            "password": password,
            "provider": "db",
            "refresh": True
        }
        
        try:
            response = self.session.post(url, json=payload)
            response.raise_for_status()
            data = response.json()
            self.access_token = data.get("access_token")
            self.refresh_token = data.get("refresh_token")
            
            # Set authorization header for future requests
            self.session.headers.update({
                "Authorization": f"Bearer {self.access_token}",
                "Content-Type": "application/json"
            })
            
            print("✅ Successfully authenticated with Superset")
            return True
        except Exception as e:
            print(f"❌ Failed to authenticate: {e}")
            return False
    
    def create_role(self, role_name: str) -> Optional[int]:
        """Create a new role in Superset"""
        url = f"{self.base_url}/api/v1/security/roles/"
        payload = {"name": role_name}
        
        try:
            response = self.session.post(url, json=payload)
            if response.status_code == 201:
                role_id = response.json().get("id")
                print(f"  ✅ Created role: {role_name} (ID: {role_id})")
                return role_id
            elif response.status_code == 409:
                # Role already exists, get its ID
                print(f"  ℹ️  Role already exists: {role_name}")
                return self.get_role_id(role_name)
            else:
                print(f"  ⚠️  Unexpected response: {response.status_code}")
                return None
        except Exception as e:
            print(f"  ❌ Failed to create role {role_name}: {e}")
            return None
    
    def get_role_id(self, role_name: str) -> Optional[int]:
        """Get role ID by name"""
        url = f"{self.base_url}/api/v1/security/roles/"
        
        try:
            response = self.session.get(url)
            response.raise_for_status()
            roles = response.json().get("result", [])
            
            for role in roles:
                if role.get("name") == role_name:
                    return role.get("id")
            return None
        except Exception as e:
            print(f"  ❌ Failed to get role ID: {e}")
            return None
    
    def list_datasets(self) -> List[Dict]:
        """List all datasets in Superset"""
        url = f"{self.base_url}/api/v1/dataset/"
        
        try:
            response = self.session.get(url)
            response.raise_for_status()
            return response.json().get("result", [])
        except Exception as e:
            print(f"❌ Failed to list datasets: {e}")
            return []
    
    def create_rls_rule(self, dataset_id: int, role_id: int, tenant_id: str, tenant_name: str) -> bool:
        """Create RLS rule for a dataset and role"""
        url = f"{self.base_url}/api/v1/rowlevelsecurity/"
        
        # RLS filter clause
        clause = f"tenant_id = '{tenant_id}'"
        
        payload = {
            "name": f"Tenant {tenant_name} - Dataset {dataset_id}",
            "description": f"Row-level security for tenant {tenant_name}",
            "filter_type": "Regular",
            "tables": [dataset_id],
            "roles": [role_id],
            "clause": clause,
            "group_key": None
        }
        
        try:
            response = self.session.post(url, json=payload)
            if response.status_code in [200, 201]:
                print(f"    ✅ Created RLS rule for dataset {dataset_id}")
                return True
            else:
                print(f"    ⚠️  RLS rule may already exist for dataset {dataset_id}")
                return False
        except Exception as e:
            print(f"    ❌ Failed to create RLS rule: {e}")
            return False
    
    def provision_tenant(self, tenant_id: str, tenant_name: str) -> bool:
        """Provision a tenant with role and RLS rules"""
        print(f"\n📦 Provisioning tenant: {tenant_name} ({tenant_id})")
        
        # Create tenant-specific role
        role_name = f"tenant_{tenant_name}_users"
        role_id = self.create_role(role_name)
        
        if not role_id:
            print(f"  ❌ Failed to create role for tenant {tenant_name}")
            return False
        
        # Get all datasets
        datasets = self.list_datasets()
        print(f"  📊 Found {len(datasets)} datasets")
        
        # Create RLS rules for each dataset
        success_count = 0
        for dataset in datasets:
            dataset_id = dataset.get("id")
            dataset_name = dataset.get("table_name")
            
            # Only create RLS for tables with tenant_id column
            # You may want to check dataset schema here
            if self.create_rls_rule(dataset_id, role_id, tenant_id, tenant_name):
                success_count += 1
        
        print(f"  ✅ Created {success_count}/{len(datasets)} RLS rules")
        return True
    
    def create_global_admin_role(self) -> bool:
        """Create global admin role with no RLS restrictions"""
        print(f"\n🔐 Creating global admin role")
        
        role_name = "Uisce Global Admin"
        role_id = self.create_role(role_name)
        
        if not role_id:
            print(f"  ❌ Failed to create global admin role")
            return False
        
        # Global admins don't need RLS rules - they see everything
        print(f"  ✅ Global admin role created (no RLS filters)")
        print(f"  ℹ️  Assign this role to Uisce organization admins")
        
        return True


def main():
    print("🔐 Superset Multi-Tenant Security Configuration")
    print("=" * 60)
    print(f"Superset URL: {SUPERSET_URL}")
    print()
    
    # Initialize manager
    manager = SupersetSecurityManager(SUPERSET_URL, SUPERSET_USERNAME, SUPERSET_PASSWORD)
    
    # Authenticate
    if not manager.login():
        print("\n❌ Authentication failed. Exiting.")
        sys.exit(1)
    
    # Create global admin role
    manager.create_global_admin_role()
    
    # Example: Provision tenants
    # In production, you would fetch tenants from your database
    print("\n" + "=" * 60)
    print("📋 Tenant Provisioning")
    print("=" * 60)
    print("\nTo provision tenants, you need to:")
    print("1. Fetch tenant list from your database")
    print("2. Call manager.provision_tenant(tenant_id, tenant_name) for each")
    print("\nExample:")
    print("  manager.provision_tenant('00000000-0000-0000-0000-000000000001', 'tenant-a')")
    print("  manager.provision_tenant('00000000-0000-0000-0000-000000000002', 'tenant-b')")
    print()
    
    # Uncomment to provision example tenants:
    # manager.provision_tenant('00000000-0000-0000-0000-000000000001', 'tenant-a')
    # manager.provision_tenant('00000000-0000-0000-0000-000000000002', 'tenant-b')
    
    print("\n✅ Superset security configuration complete!")
    print("\nNext steps:")
    print("1. Assign users to tenant-specific roles in Superset")
    print("2. Assign Uisce admins to 'Uisce Global Admin' role")
    print("3. Test dashboard access with different users")


if __name__ == "__main__":
    main()
