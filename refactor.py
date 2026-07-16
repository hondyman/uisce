import os
import re

for root, dirs, files in os.walk('backend/internal'):
    for file in files:
        if file.endswith('.go'):
            path = os.path.join(root, file)
            with open(path, 'r') as f:
                content = f.read()
            
            new_content = content
            
            # Signatures
            new_content = re.sub(r'GetBODefinition\(boID, tenantID string\)', r'GetBODefinition(boID, tenantID, instanceID string)', new_content)
            new_content = re.sub(r'GetBODefinition\(_, _ string\)', r'GetBODefinition(_, _, _ string)', new_content)
            
            # Calls where it's exactly 2 args
            # We will use a regex that matches GetBODefinition(arg1, arg2)
            # This is risky, but we can compile after and fix.
            # Let's replace known calls:
            new_content = new_content.replace('h.Repo.GetBODefinition(boID, tenantID)', 'h.Repo.GetBODefinition(boID, tenantID, instanceID)')
            new_content = new_content.replace('r.GetBODefinition(boID, tenantID)', 'r.GetBODefinition(boID, tenantID, instanceID)')
            new_content = new_content.replace('g.BORepository.GetBODefinition(req.BusinessObjectID, req.TenantID)', 'g.BORepository.GetBODefinition(req.BusinessObjectID, req.TenantID, "")')
            new_content = new_content.replace('g.BORepository.GetBODefinition(foundField.ReferenceBOID, genCtx.Request.TenantID)', 'g.BORepository.GetBODefinition(foundField.ReferenceBOID, genCtx.Request.TenantID, "")')
            new_content = new_content.replace('g.BORepository.GetBODefinition(targetBOID, genCtx.Request.TenantID)', 'g.BORepository.GetBODefinition(targetBOID, genCtx.Request.TenantID, "")')
            new_content = new_content.replace('s.GetBODefinition("", "")', 's.GetBODefinition("", "", "")')
            new_content = new_content.replace('c.repo.GetBODefinition(boID, tenantID)', 'c.repo.GetBODefinition(boID, tenantID, instanceID)')
            new_content = new_content.replace('cached.GetBODefinition("bo-1", "tenant-a")', 'cached.GetBODefinition("bo-1", "tenant-a", "")')
            new_content = new_content.replace('cached.GetBODefinition("bo-1", "tenant-b")', 'cached.GetBODefinition("bo-1", "tenant-b", "")')
            new_content = new_content.replace('repo.GetBODefinition(boID, tenantID)', 'repo.GetBODefinition(boID, tenantID, "")')

            if new_content != content:
                with open(path, 'w') as f:
                    f.write(new_content)
                print("Updated", path)
