import React, { useState } from "react";
import { Card, Table, Spinner } from "../";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../../api";

export function SSOConfig() {
  const queryClient = useQueryClient();
  const [isAdding, setIsAdding] = useState(false);

  // In a real app, this would fetch from /api/v1/admin/sso-providers
  // For now we'll query Hasura directly or use our new handlers
  const { data: providers, isLoading } = useQuery({
    queryKey: ["sso-providers"],
    queryFn: () => api<{ data: any[] }>("/admin/sso-providers"), // Need to ensure this endpoint exists
  });

  const columns = ["Name", "Type", "Status", "Primary", "Actions"];
  const rows = providers?.data?.map((p) => [
    p.provider_name,
    p.provider_type.toUpperCase(),
    p.is_active ? "Active" : "Disabled",
    p.is_primary ? "Yes" : "No",
    <button className="btn btn-sm btn-outline">Edit</button>
  ]) || [];

  return (
    <div className="sso-config">
      <div className="section-header">
        <h2 className="text-xl font-semibold">SSO Providers</h2>
        <button 
          className="btn btn-primary"
          onClick={() => setIsAdding(true)}
        >
          + Add Provider
        </button>
      </div>

      <Card className="mt-4">
        {isLoading ? (
          <Spinner size="md" />
        ) : (
          <Table
            columns={columns}
            rows={rows}
            empty="No SSO providers configured"
          />
        )}
      </Card>

      {isAdding && (
         <div className="mt-8">
            <h3 className="text-lg font-medium">Add New SSO Provider</h3>
            <Card className="mt-2">
                <form className="space-y-4">
                    <div className="grid grid-2 gap-4">
                        <div className="field">
                            <label>Provider Name</label>
                            <input type="text" placeholder="Okta, Azure AD, etc." />
                        </div>
                        <div className="field">
                            <label>Type</label>
                            <select>
                                <option value="saml">SAML 2.0</option>
                                <option value="oidc">OIDC</option>
                            </select>
                        </div>
                    </div>
                    
                    <div className="field">
                        <label>Metadata URL / Issuer</label>
                        <input type="text" placeholder="https://..." />
                    </div>

                    <div className="actions">
                        <button type="button" className="btn btn-outline" onClick={() => setIsAdding(false)}>Cancel</button>
                        <button type="submit" className="btn btn-primary">Save Provider</button>
                    </div>
                </form>
            </Card>
         </div>
      )}
    </div>
  );
}
