# Quick Tenant Scope Bookmarklets

These bookmarklets let you instantly seed tenant scope with one click.

## How to Install

1. Create a new bookmark in your browser
2. Name it "Seed Northwind Tenant"
3. For the URL, paste the code below (starts with `javascript:`)
4. Click the bookmark when on the Fabric Builder app
5. Reload the page

## Northwind Tenant (Default)

```javascript
javascript:(function(){const t={id:"910638ba-a459-4a3f-bb2d-78391b0595f6",display_name:"Northwind",name:"Northwind"};const d={id:"982aef38-418f-46dc-acd0-35fe8f3b97b0",alpha_datasource_id:"982aef38-418f-46dc-acd0-35fe8f3b97b0",source_name:"Northwind Database",alpha_datasource:{datasource_name:"Northwind Database"}};localStorage.setItem("selected_tenant",JSON.stringify(t));localStorage.setItem("selected_datasource",JSON.stringify(d));alert("✅ Northwind tenant scope set!\n\nReload the page to activate.");})();
```

## GOLD_COPY Tenant (Alternative)

```javascript
javascript:(function(){const t={id:"c52a4906-6177-44a6-80c6-0c1b7c5f30b3",display_name:"GOLD_COPY",name:"GOLD_COPY"};const d={id:"f938c8e6-6e11-405c-a700-ce5eacc5f45b",alpha_datasource_id:"f938c8e6-6e11-405c-a700-ce5eacc5f45b",source_name:"GOLD_COPY Database",alpha_datasource:{datasource_name:"GOLD_COPY Database"}};localStorage.setItem("selected_tenant",JSON.stringify(t));localStorage.setItem("selected_datasource",JSON.stringify(d));alert("✅ GOLD_COPY tenant scope set!\n\nReload the page to activate.");})();
```

## Clear Tenant Scope

```javascript
javascript:(function(){localStorage.removeItem("selected_tenant");localStorage.removeItem("selected_datasource");localStorage.removeItem("selected_product");alert("❌ Tenant scope cleared!\n\nReload the page.");})();
```

## For Safari Users

Safari might not allow bookmarklets with `javascript:` URLs by default. Use the console script instead:

1. Open DevTools Console (Cmd+Option+J)
2. Paste the contents of `seed_tenant_scope.js`
3. Press Enter
4. Reload the page

## For Chrome/Firefox Users

These bookmarklets work directly:
1. Right-click bookmarks bar → "Add page..."
2. Name: "Seed Northwind"
3. URL: Paste the javascript code above
4. Save
5. Click when needed
6. Reload page
