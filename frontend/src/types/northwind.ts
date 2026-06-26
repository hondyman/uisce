/**
 * Northwind Database Business Objects (BOs)
 * Complete TypeScript definitions for the 8 core BOs with all subtypes and fields
 * Based on Microsoft's Northwind sample database
 * 
 * This model demonstrates:
 * - Core BO structure (template)
 * - Subtypes (e.g., VIP Customer)
 * - Inheritance (fields cloned from parent)
 * - Custom fields (extensibility)
 * - Instances (individual records)
 */

// ============================================================================
// FIELD & SUBTYPE INTERFACES
// ============================================================================

export interface FieldDefinition {
  key: string;
  name: string;
  displayName: string; // Business-friendly name (e.g., "Customer Name")
  technicalName: string; // DB column name (e.g., "customer_name")
  type: 'text' | 'email' | 'number' | 'date' | 'datetime' | 'boolean' | 'currency' | 'json' | 'array' | 'image' | 'reference';
  isCore: boolean; // true if from Northwind core
  required?: boolean;
  description?: string;
  referenceEntity?: string; // if type='reference', which entity it references
  isSystem?: boolean; // if true, user cannot delete
  sequence?: number; // UI display order
  createdAt?: string;
  createdBy?: string;
  lastModifiedAt?: string;
  lastModifiedBy?: string;
}

export interface SubtypeDefinition {
  key: string;
  name: string;
  displayName: string;
  technicalName: string;
  description?: string;
  subtypeFields: FieldDefinition[];
  isCore: boolean;
  basedOnEntity?: string; // if cloned
  cloneParentKey?: string;
  createdAt?: string;
  createdBy?: string;
}

// ============================================================================
// BO DEFINITION INTERFACE
// ============================================================================

export interface BusinessObjectDefinition {
  key: string;
  name: string; // Display name (e.g., "Customer")
  displayName: string;
  technicalName: string; // DB table name (e.g., "customers")
  description?: string;
  icon?: string;
  
  // Core vs Custom
  isCore: boolean; // true if Northwind core BO
  coreFields: FieldDefinition[]; // explicitly separate core fields
  customFields: FieldDefinition[]; // user-added custom fields
  
  // Subtypes
  subtypes: Record<string, SubtypeDefinition>;
  
  // Clone tracking
  clonesFrom?: string; // if cloned from another BO
  cloneParentKey?: string;
  cloneParentDisplayName?: string;
  
  // Metadata
  category?: string; // e.g., "Sales", "Inventory", "HR"
  instanceCount?: number; // runtime stats
  createdAt?: string;
  createdBy?: string;
  lastModifiedAt?: string;
  lastModifiedBy?: string;
}

export interface BusinessObjectsRegistry {
  [key: string]: BusinessObjectDefinition;
}

// ============================================================================
// INSTANCE INTERFACE (Individual Records)
// ============================================================================

export interface BusinessObjectInstance {
  id: string; // UUID
  businessObjectKey: string; // e.g., "customer"
  subtypeKey?: string; // e.g., "vip_customer"
  tenantId: string;
  datasourceId: string;
  
  // Core fields (values)
  coreFieldValues: Record<string, any>;
  
  // Custom fields (extensible)
  customFieldValues: Record<string, any>;
  
  // Metadata
  createdAt: string;
  createdBy: string;
  lastModifiedAt: string;
  lastModifiedBy: string;
  isDeleted?: boolean;
  deletedAt?: string;
}

// ============================================================================
// NORTHWIND: 1. CUSTOMER BO
// ============================================================================

export const CUSTOMER_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'customer_id',
    name: 'Customer ID',
    displayName: 'ID',
    technicalName: 'customer_id',
    type: 'text',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'company_name',
    name: 'Company Name',
    displayName: 'Company',
    technicalName: 'company_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'contact_name',
    name: 'Contact Name',
    displayName: 'Primary Contact',
    technicalName: 'contact_name',
    type: 'text',
    isCore: true,
    sequence: 2,
  },
  {
    key: 'contact_title',
    name: 'Contact Title',
    displayName: 'Contact Position',
    technicalName: 'contact_title',
    type: 'text',
    isCore: true,
    sequence: 3,
  },
  {
    key: 'address',
    name: 'Address',
    displayName: 'Street Address',
    technicalName: 'address',
    type: 'text',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'city',
    name: 'City',
    displayName: 'City',
    technicalName: 'city',
    type: 'text',
    isCore: true,
    sequence: 5,
  },
  {
    key: 'region',
    name: 'Region',
    displayName: 'State/Province',
    technicalName: 'region',
    type: 'text',
    isCore: true,
    sequence: 6,
  },
  {
    key: 'postal_code',
    name: 'Postal Code',
    displayName: 'ZIP/Postal Code',
    technicalName: 'postal_code',
    type: 'text',
    isCore: true,
    sequence: 7,
  },
  {
    key: 'country',
    name: 'Country',
    displayName: 'Country',
    technicalName: 'country',
    type: 'text',
    isCore: true,
    sequence: 8,
  },
  {
    key: 'phone',
    name: 'Phone',
    displayName: 'Phone Number',
    technicalName: 'phone',
    type: 'text',
    isCore: true,
    sequence: 9,
  },
  {
    key: 'fax',
    name: 'Fax',
    displayName: 'Fax Number',
    technicalName: 'fax',
    type: 'text',
    isCore: true,
    sequence: 10,
  },
];

export const CUSTOMER_SUBTYPES: Record<string, SubtypeDefinition> = {
  standard_customer: {
    key: 'standard_customer',
    name: 'Standard Customer',
    displayName: 'Standard',
    technicalName: 'standard_customer',
    description: 'Regular business customer with standard terms',
    subtypeFields: [],
    isCore: true,
  },
  vip_customer: {
    key: 'vip_customer',
    name: 'VIP Customer',
    displayName: 'VIP',
    technicalName: 'vip_customer',
    description: 'High-value customer with premium terms',
    subtypeFields: [
      {
        key: 'vip_tier',
        name: 'VIP Tier',
        displayName: 'VIP Level',
        technicalName: 'vip_tier',
        type: 'text',
        isCore: true,
        sequence: 0,
      },
      {
        key: 'discount_percentage',
        name: 'Discount %',
        displayName: 'Discount Percentage',
        technicalName: 'discount_percentage',
        type: 'number',
        isCore: true,
        sequence: 1,
      },
    ],
    isCore: true,
  },
};

export const CUSTOMER_BO: BusinessObjectDefinition = {
  key: 'customer',
  name: 'Customer',
  displayName: 'Customers',
  technicalName: 'customers',
  description: 'Customers and their demographics, linking to orders and demographics',
  icon: 'users',
  isCore: true,
  coreFields: CUSTOMER_CORE_FIELDS,
  customFields: [],
  subtypes: CUSTOMER_SUBTYPES,
  category: 'Sales',
};

// ============================================================================
// NORTHWIND: 2. EMPLOYEE BO
// ============================================================================

export const EMPLOYEE_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'employee_id',
    name: 'Employee ID',
    displayName: 'ID',
    technicalName: 'employee_id',
    type: 'number',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'last_name',
    name: 'Last Name',
    displayName: 'Last Name',
    technicalName: 'last_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'first_name',
    name: 'First Name',
    displayName: 'First Name',
    technicalName: 'first_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 2,
  },
  {
    key: 'title',
    name: 'Title',
    displayName: 'Job Title',
    technicalName: 'title',
    type: 'text',
    isCore: true,
    sequence: 3,
  },
  {
    key: 'title_of_courtesy',
    name: 'Title of Courtesy',
    displayName: 'Courtesy Title',
    technicalName: 'title_of_courtesy',
    type: 'text',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'birth_date',
    name: 'Birth Date',
    displayName: 'Date of Birth',
    technicalName: 'birth_date',
    type: 'date',
    isCore: true,
    sequence: 5,
  },
  {
    key: 'hire_date',
    name: 'Hire Date',
    displayName: 'Hire Date',
    technicalName: 'hire_date',
    type: 'date',
    isCore: true,
    sequence: 6,
  },
  {
    key: 'address',
    name: 'Address',
    displayName: 'Street Address',
    technicalName: 'address',
    type: 'text',
    isCore: true,
    sequence: 7,
  },
  {
    key: 'city',
    name: 'City',
    displayName: 'City',
    technicalName: 'city',
    type: 'text',
    isCore: true,
    sequence: 8,
  },
  {
    key: 'region',
    name: 'Region',
    displayName: 'State/Province',
    technicalName: 'region',
    type: 'text',
    isCore: true,
    sequence: 9,
  },
  {
    key: 'postal_code',
    name: 'Postal Code',
    displayName: 'ZIP/Postal Code',
    technicalName: 'postal_code',
    type: 'text',
    isCore: true,
    sequence: 10,
  },
  {
    key: 'country',
    name: 'Country',
    displayName: 'Country',
    technicalName: 'country',
    type: 'text',
    isCore: true,
    sequence: 11,
  },
  {
    key: 'home_phone',
    name: 'Home Phone',
    displayName: 'Phone Number',
    technicalName: 'home_phone',
    type: 'text',
    isCore: true,
    sequence: 12,
  },
  {
    key: 'extension',
    name: 'Extension',
    displayName: 'Phone Extension',
    technicalName: 'extension',
    type: 'text',
    isCore: true,
    sequence: 13,
  },
  {
    key: 'photo',
    name: 'Photo',
    displayName: 'Employee Photo',
    technicalName: 'photo',
    type: 'image',
    isCore: true,
    sequence: 14,
  },
  {
    key: 'notes',
    name: 'Notes',
    displayName: 'Notes',
    technicalName: 'notes',
    type: 'text',
    isCore: true,
    sequence: 15,
  },
  {
    key: 'reports_to',
    name: 'Reports To',
    displayName: 'Manager',
    technicalName: 'reports_to',
    type: 'reference',
    referenceEntity: 'employee',
    isCore: true,
    sequence: 16,
  },
];

export const EMPLOYEE_SUBTYPES: Record<string, SubtypeDefinition> = {
  employee: {
    key: 'employee',
    name: 'Employee',
    displayName: 'Standard Employee',
    technicalName: 'employee',
    description: 'Regular employee',
    subtypeFields: [],
    isCore: true,
  },
  sales_representative: {
    key: 'sales_representative',
    name: 'Sales Representative',
    displayName: 'Sales Rep',
    technicalName: 'sales_representative',
    description: 'Employee with assigned territories',
    subtypeFields: [
      {
        key: 'territories',
        name: 'Assigned Territories',
        displayName: 'Territories',
        technicalName: 'territories',
        type: 'array',
        isCore: true,
        referenceEntity: 'territory',
        sequence: 0,
      },
      {
        key: 'sales_quota',
        name: 'Sales Quota',
        displayName: 'Annual Quota',
        technicalName: 'sales_quota',
        type: 'currency',
        isCore: true,
        sequence: 1,
      },
    ],
    isCore: true,
  },
  manager: {
    key: 'manager',
    name: 'Manager',
    displayName: 'Management',
    technicalName: 'manager',
    description: 'Employee who manages other employees',
    subtypeFields: [
      {
        key: 'direct_reports',
        name: 'Direct Reports',
        displayName: 'Team Size',
        technicalName: 'direct_reports',
        type: 'array',
        isCore: true,
        referenceEntity: 'employee',
        sequence: 0,
      },
      {
        key: 'budget',
        name: 'Budget',
        displayName: 'Department Budget',
        technicalName: 'budget',
        type: 'currency',
        isCore: true,
        sequence: 1,
      },
    ],
    isCore: true,
  },
};

export const EMPLOYEE_BO: BusinessObjectDefinition = {
  key: 'employee',
  name: 'Employee',
  displayName: 'Employees',
  technicalName: 'employees',
  description: 'Staff hierarchy and territories, supporting order assignments',
  icon: 'users',
  isCore: true,
  coreFields: EMPLOYEE_CORE_FIELDS,
  customFields: [],
  subtypes: EMPLOYEE_SUBTYPES,
  category: 'HR',
};

// ============================================================================
// NORTHWIND: 3. SUPPLIER BO
// ============================================================================

export const SUPPLIER_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'supplier_id',
    name: 'Supplier ID',
    displayName: 'ID',
    technicalName: 'supplier_id',
    type: 'number',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'company_name',
    name: 'Company Name',
    displayName: 'Company',
    technicalName: 'company_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'contact_name',
    name: 'Contact Name',
    displayName: 'Primary Contact',
    technicalName: 'contact_name',
    type: 'text',
    isCore: true,
    sequence: 2,
  },
  {
    key: 'contact_title',
    name: 'Contact Title',
    displayName: 'Contact Position',
    technicalName: 'contact_title',
    type: 'text',
    isCore: true,
    sequence: 3,
  },
  {
    key: 'address',
    name: 'Address',
    displayName: 'Street Address',
    technicalName: 'address',
    type: 'text',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'city',
    name: 'City',
    displayName: 'City',
    technicalName: 'city',
    type: 'text',
    isCore: true,
    sequence: 5,
  },
  {
    key: 'region',
    name: 'Region',
    displayName: 'State/Province',
    technicalName: 'region',
    type: 'text',
    isCore: true,
    sequence: 6,
  },
  {
    key: 'postal_code',
    name: 'Postal Code',
    displayName: 'ZIP/Postal Code',
    technicalName: 'postal_code',
    type: 'text',
    isCore: true,
    sequence: 7,
  },
  {
    key: 'country',
    name: 'Country',
    displayName: 'Country',
    technicalName: 'country',
    type: 'text',
    isCore: true,
    sequence: 8,
  },
  {
    key: 'phone',
    name: 'Phone',
    displayName: 'Phone Number',
    technicalName: 'phone',
    type: 'text',
    isCore: true,
    sequence: 9,
  },
  {
    key: 'fax',
    name: 'Fax',
    displayName: 'Fax Number',
    technicalName: 'fax',
    type: 'text',
    isCore: true,
    sequence: 10,
  },
  {
    key: 'home_page',
    name: 'Home Page',
    displayName: 'Website',
    technicalName: 'home_page',
    type: 'text',
    isCore: true,
    sequence: 11,
  },
];

export const SUPPLIER_SUBTYPES: Record<string, SubtypeDefinition> = {
  supplier: {
    key: 'supplier',
    name: 'Supplier',
    displayName: 'Standard',
    technicalName: 'supplier',
    description: 'Regular supplier',
    subtypeFields: [],
    isCore: true,
  },
  domestic_supplier: {
    key: 'domestic_supplier',
    name: 'Domestic Supplier',
    displayName: 'Domestic',
    technicalName: 'domestic_supplier',
    description: 'Supplier based in home country',
    subtypeFields: [
      {
        key: 'state_license',
        name: 'State License',
        displayName: 'License Number',
        technicalName: 'state_license',
        type: 'text',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  international_supplier: {
    key: 'international_supplier',
    name: 'International Supplier',
    displayName: 'International',
    technicalName: 'international_supplier',
    description: 'Supplier based outside home country',
    subtypeFields: [
      {
        key: 'tariff_code',
        name: 'Tariff Code',
        displayName: 'HS Code',
        technicalName: 'tariff_code',
        type: 'text',
        isCore: true,
        sequence: 0,
      },
      {
        key: 'payment_terms',
        name: 'Payment Terms',
        displayName: 'Terms',
        technicalName: 'payment_terms',
        type: 'text',
        isCore: true,
        sequence: 1,
      },
    ],
    isCore: true,
  },
};

export const SUPPLIER_BO: BusinessObjectDefinition = {
  key: 'supplier',
  name: 'Supplier',
  displayName: 'Suppliers',
  technicalName: 'suppliers',
  description: 'Tracks vendors for product sourcing',
  icon: 'truck',
  isCore: true,
  coreFields: SUPPLIER_CORE_FIELDS,
  customFields: [],
  subtypes: SUPPLIER_SUBTYPES,
  category: 'Procurement',
};

// ============================================================================
// NORTHWIND: 4. PRODUCT BO
// ============================================================================

export const PRODUCT_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'product_id',
    name: 'Product ID',
    displayName: 'ID',
    technicalName: 'product_id',
    type: 'number',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'product_name',
    name: 'Product Name',
    displayName: 'Name',
    technicalName: 'product_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'supplier_id',
    name: 'Supplier ID',
    displayName: 'Supplier',
    technicalName: 'supplier_id',
    type: 'reference',
    referenceEntity: 'supplier',
    isCore: true,
    sequence: 2,
  },
  {
    key: 'category_id',
    name: 'Category ID',
    displayName: 'Category',
    technicalName: 'category_id',
    type: 'text',
    isCore: true,
    sequence: 3,
  },
  {
    key: 'quantity_per_unit',
    name: 'Quantity Per Unit',
    displayName: 'Unit Size',
    technicalName: 'quantity_per_unit',
    type: 'text',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'unit_price',
    name: 'Unit Price',
    displayName: 'Price',
    technicalName: 'unit_price',
    type: 'currency',
    isCore: true,
    sequence: 5,
  },
  {
    key: 'units_in_stock',
    name: 'Units In Stock',
    displayName: 'Stock Level',
    technicalName: 'units_in_stock',
    type: 'number',
    isCore: true,
    sequence: 6,
  },
  {
    key: 'units_on_order',
    name: 'Units On Order',
    displayName: 'On Order',
    technicalName: 'units_on_order',
    type: 'number',
    isCore: true,
    sequence: 7,
  },
  {
    key: 'reorder_level',
    name: 'Reorder Level',
    displayName: 'Min Qty',
    technicalName: 'reorder_level',
    type: 'number',
    isCore: true,
    sequence: 8,
  },
  {
    key: 'discontinued',
    name: 'Discontinued',
    displayName: 'Discontinued',
    technicalName: 'discontinued',
    type: 'boolean',
    isCore: true,
    sequence: 9,
  },
  {
    key: 'description',
    name: 'Description',
    displayName: 'Description',
    technicalName: 'description',
    type: 'text',
    isCore: true,
    sequence: 10,
  },
];

export const PRODUCT_SUBTYPES: Record<string, SubtypeDefinition> = {
  beverage: {
    key: 'beverage',
    name: 'Beverage',
    displayName: 'Beverages',
    technicalName: 'beverage',
    description: 'Beverage products (Category 1)',
    subtypeFields: [
      {
        key: 'alcohol_content',
        name: 'Alcohol Content',
        displayName: 'ABV',
        technicalName: 'alcohol_content',
        type: 'number',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  condiment: {
    key: 'condiment',
    name: 'Condiment',
    displayName: 'Condiments',
    technicalName: 'condiment',
    description: 'Condiment products (Category 2)',
    subtypeFields: [],
    isCore: true,
  },
  confection: {
    key: 'confection',
    name: 'Confection',
    displayName: 'Confections',
    technicalName: 'confection',
    description: 'Confection products (Category 3)',
    subtypeFields: [],
    isCore: true,
  },
  dairy: {
    key: 'dairy',
    name: 'Dairy',
    displayName: 'Dairy',
    technicalName: 'dairy',
    description: 'Dairy products (Category 4)',
    subtypeFields: [
      {
        key: 'shelf_life_days',
        name: 'Shelf Life (Days)',
        displayName: 'Expiration',
        technicalName: 'shelf_life_days',
        type: 'number',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  grains_cereals: {
    key: 'grains_cereals',
    name: 'Grains/Cereals',
    displayName: 'Grains',
    technicalName: 'grains_cereals',
    description: 'Grain and cereal products (Category 5)',
    subtypeFields: [],
    isCore: true,
  },
  meat_poultry: {
    key: 'meat_poultry',
    name: 'Meat/Poultry',
    displayName: 'Meat',
    technicalName: 'meat_poultry',
    description: 'Meat and poultry products (Category 6)',
    subtypeFields: [
      {
        key: 'storage_temperature',
        name: 'Storage Temp (C)',
        displayName: 'Temperature',
        technicalName: 'storage_temperature',
        type: 'number',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  produce: {
    key: 'produce',
    name: 'Produce',
    displayName: 'Produce',
    technicalName: 'produce',
    description: 'Produce products (Category 7)',
    subtypeFields: [
      {
        key: 'harvest_date',
        name: 'Harvest Date',
        displayName: 'Harvest',
        technicalName: 'harvest_date',
        type: 'date',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  seafood: {
    key: 'seafood',
    name: 'Seafood',
    displayName: 'Seafood',
    technicalName: 'seafood',
    description: 'Seafood products (Category 8)',
    subtypeFields: [
      {
        key: 'catch_date',
        name: 'Catch Date',
        displayName: 'Caught',
        technicalName: 'catch_date',
        type: 'date',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
};

export const PRODUCT_BO: BusinessObjectDefinition = {
  key: 'product',
  name: 'Product',
  displayName: 'Products',
  technicalName: 'products',
  description: 'Manages inventory items, categories, and pricing',
  icon: 'box',
  isCore: true,
  coreFields: PRODUCT_CORE_FIELDS,
  customFields: [],
  subtypes: PRODUCT_SUBTYPES,
  category: 'Inventory',
};

// ============================================================================
// NORTHWIND: 5. ORDER BO
// ============================================================================

export const ORDER_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'order_id',
    name: 'Order ID',
    displayName: 'ID',
    technicalName: 'order_id',
    type: 'number',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'customer_id',
    name: 'Customer ID',
    displayName: 'Customer',
    technicalName: 'customer_id',
    type: 'reference',
    referenceEntity: 'customer',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'employee_id',
    name: 'Employee ID',
    displayName: 'Sales Rep',
    technicalName: 'employee_id',
    type: 'reference',
    referenceEntity: 'employee',
    isCore: true,
    sequence: 2,
  },
  {
    key: 'order_date',
    name: 'Order Date',
    displayName: 'Order Date',
    technicalName: 'order_date',
    type: 'datetime',
    isCore: true,
    required: true,
    sequence: 3,
  },
  {
    key: 'required_date',
    name: 'Required Date',
    displayName: 'Due Date',
    technicalName: 'required_date',
    type: 'datetime',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'shipped_date',
    name: 'Shipped Date',
    displayName: 'Shipped Date',
    technicalName: 'shipped_date',
    type: 'datetime',
    isCore: true,
    sequence: 5,
  },
  {
    key: 'ship_via',
    name: 'Ship Via',
    displayName: 'Shipper',
    technicalName: 'ship_via',
    type: 'reference',
    referenceEntity: 'shipper',
    isCore: true,
    sequence: 6,
  },
  {
    key: 'freight',
    name: 'Freight',
    displayName: 'Shipping Cost',
    technicalName: 'freight',
    type: 'currency',
    isCore: true,
    sequence: 7,
  },
  {
    key: 'ship_name',
    name: 'Ship Name',
    displayName: 'Ship To',
    technicalName: 'ship_name',
    type: 'text',
    isCore: true,
    sequence: 8,
  },
  {
    key: 'ship_address',
    name: 'Ship Address',
    displayName: 'Address',
    technicalName: 'ship_address',
    type: 'text',
    isCore: true,
    sequence: 9,
  },
  {
    key: 'ship_city',
    name: 'Ship City',
    displayName: 'City',
    technicalName: 'ship_city',
    type: 'text',
    isCore: true,
    sequence: 10,
  },
  {
    key: 'ship_region',
    name: 'Ship Region',
    displayName: 'State',
    technicalName: 'ship_region',
    type: 'text',
    isCore: true,
    sequence: 11,
  },
  {
    key: 'ship_postal_code',
    name: 'Ship Postal Code',
    displayName: 'ZIP',
    technicalName: 'ship_postal_code',
    type: 'text',
    isCore: true,
    sequence: 12,
  },
  {
    key: 'ship_country',
    name: 'Ship Country',
    displayName: 'Country',
    technicalName: 'ship_country',
    type: 'text',
    isCore: true,
    sequence: 13,
  },
];

export const ORDER_SUBTYPES: Record<string, SubtypeDefinition> = {
  standard_order: {
    key: 'standard_order',
    name: 'Standard Order',
    displayName: 'Standard',
    technicalName: 'standard_order',
    description: 'Regular order with standard processing',
    subtypeFields: [],
    isCore: true,
  },
  rush_order: {
    key: 'rush_order',
    name: 'Rush Order',
    displayName: 'Rush',
    technicalName: 'rush_order',
    description: 'Expedited order (shipped before required date)',
    subtypeFields: [
      {
        key: 'rush_fee',
        name: 'Rush Fee',
        displayName: 'Expedite Fee',
        technicalName: 'rush_fee',
        type: 'currency',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  backorder: {
    key: 'backorder',
    name: 'Backorder',
    displayName: 'Backorder',
    technicalName: 'backorder',
    description: 'Order pending shipment (not yet shipped)',
    subtypeFields: [
      {
        key: 'expected_ship_date',
        name: 'Expected Ship Date',
        displayName: 'Est. Ship',
        technicalName: 'expected_ship_date',
        type: 'date',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
};

export const ORDER_BO: BusinessObjectDefinition = {
  key: 'order',
  name: 'Order',
  displayName: 'Orders',
  technicalName: 'orders',
  description: 'Captures sales transactions, shipping, and customer details',
  icon: 'shopping-cart',
  isCore: true,
  coreFields: ORDER_CORE_FIELDS,
  customFields: [],
  subtypes: ORDER_SUBTYPES,
  category: 'Sales',
};

// ============================================================================
// NORTHWIND: 6. ORDER DETAIL BO
// ============================================================================

export const ORDER_DETAIL_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'order_id',
    name: 'Order ID',
    displayName: 'Order',
    technicalName: 'order_id',
    type: 'reference',
    referenceEntity: 'order',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'product_id',
    name: 'Product ID',
    displayName: 'Product',
    technicalName: 'product_id',
    type: 'reference',
    referenceEntity: 'product',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'unit_price',
    name: 'Unit Price',
    displayName: 'Price',
    technicalName: 'unit_price',
    type: 'currency',
    isCore: true,
    required: true,
    sequence: 2,
  },
  {
    key: 'quantity',
    name: 'Quantity',
    displayName: 'Qty',
    technicalName: 'quantity',
    type: 'number',
    isCore: true,
    required: true,
    sequence: 3,
  },
  {
    key: 'discount',
    name: 'Discount',
    displayName: 'Discount %',
    technicalName: 'discount',
    type: 'number',
    isCore: true,
    sequence: 4,
  },
  {
    key: 'extended_price',
    name: 'Extended Price',
    displayName: 'Line Total',
    technicalName: 'extended_price',
    type: 'currency',
    isCore: true,
    isSystem: true, // calculated
    sequence: 5,
  },
];

export const ORDER_DETAIL_SUBTYPES: Record<string, SubtypeDefinition> = {
  order_detail: {
    key: 'order_detail',
    name: 'Order Detail',
    displayName: 'Line Item',
    technicalName: 'order_detail',
    description: 'Order line item',
    subtypeFields: [],
    isCore: true,
  },
  bulk_line: {
    key: 'bulk_line',
    name: 'Bulk Line',
    displayName: 'Bulk',
    technicalName: 'bulk_line',
    description: 'Quantity > 10',
    subtypeFields: [
      {
        key: 'bulk_discount',
        name: 'Bulk Discount',
        displayName: 'Bulk Rate',
        technicalName: 'bulk_discount',
        type: 'number',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
  discounted_line: {
    key: 'discounted_line',
    name: 'Discounted Line',
    displayName: 'Discounted',
    technicalName: 'discounted_line',
    description: 'Line with discount applied',
    subtypeFields: [],
    isCore: true,
  },
};

export const ORDER_DETAIL_BO: BusinessObjectDefinition = {
  key: 'order_detail',
  name: 'Order Detail',
  displayName: 'Order Details',
  technicalName: 'order_details',
  description: 'Line items within orders',
  icon: 'list',
  isCore: true,
  coreFields: ORDER_DETAIL_CORE_FIELDS,
  customFields: [],
  subtypes: ORDER_DETAIL_SUBTYPES,
  category: 'Sales',
};

// ============================================================================
// NORTHWIND: 7. SHIPPER BO
// ============================================================================

export const SHIPPER_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'shipper_id',
    name: 'Shipper ID',
    displayName: 'ID',
    technicalName: 'shipper_id',
    type: 'number',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'company_name',
    name: 'Company Name',
    displayName: 'Name',
    technicalName: 'company_name',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'phone',
    name: 'Phone',
    displayName: 'Phone',
    technicalName: 'phone',
    type: 'text',
    isCore: true,
    sequence: 2,
  },
];

export const SHIPPER_SUBTYPES: Record<string, SubtypeDefinition> = {
  shipper: {
    key: 'shipper',
    name: 'Shipper',
    displayName: 'Standard',
    technicalName: 'shipper',
    description: 'Logistics provider',
    subtypeFields: [],
    isCore: true,
  },
};

export const SHIPPER_BO: BusinessObjectDefinition = {
  key: 'shipper',
  name: 'Shipper',
  displayName: 'Shippers',
  technicalName: 'shippers',
  description: 'Logistics providers for order fulfillment',
  icon: 'truck',
  isCore: true,
  coreFields: SHIPPER_CORE_FIELDS,
  customFields: [],
  subtypes: SHIPPER_SUBTYPES,
  category: 'Logistics',
};

// ============================================================================
// NORTHWIND: 8. TERRITORY/REGION BO
// ============================================================================

export const TERRITORY_CORE_FIELDS: FieldDefinition[] = [
  {
    key: 'territory_id',
    name: 'Territory ID',
    displayName: 'ID',
    technicalName: 'territory_id',
    type: 'text',
    isCore: true,
    required: true,
    isSystem: true,
    sequence: 0,
  },
  {
    key: 'territory_description',
    name: 'Territory Description',
    displayName: 'Name',
    technicalName: 'territory_description',
    type: 'text',
    isCore: true,
    required: true,
    sequence: 1,
  },
  {
    key: 'region_id',
    name: 'Region ID',
    displayName: 'Region',
    technicalName: 'region_id',
    type: 'text',
    isCore: true,
    sequence: 2,
  },
  {
    key: 'sales_representatives',
    name: 'Sales Representatives',
    displayName: 'Assigned Reps',
    technicalName: 'sales_representatives',
    type: 'array',
    referenceEntity: 'employee',
    isCore: true,
    sequence: 3,
  },
];

export const TERRITORY_SUBTYPES: Record<string, SubtypeDefinition> = {
  territory: {
    key: 'territory',
    name: 'Territory',
    displayName: 'Territory',
    technicalName: 'territory',
    description: 'Granular geographic area',
    subtypeFields: [],
    isCore: true,
  },
  region: {
    key: 'region',
    name: 'Region',
    displayName: 'Region',
    technicalName: 'region',
    description: 'High-level geographic area',
    subtypeFields: [
      {
        key: 'region_description',
        name: 'Region Description',
        displayName: 'Name',
        technicalName: 'region_description',
        type: 'text',
        isCore: true,
        sequence: 0,
      },
    ],
    isCore: true,
  },
};

export const TERRITORY_BO: BusinessObjectDefinition = {
  key: 'territory',
  name: 'Territory',
  displayName: 'Territories',
  technicalName: 'territories',
  description: 'Geographic segmentation for employees and sales',
  icon: 'map',
  isCore: true,
  coreFields: TERRITORY_CORE_FIELDS,
  customFields: [],
  subtypes: TERRITORY_SUBTYPES,
  category: 'Geography',
};

// ============================================================================
// REGISTRY: All Northwind BOs
// ============================================================================

export const NORTHWIND_BOS_REGISTRY: BusinessObjectsRegistry = {
  customer: CUSTOMER_BO,
  employee: EMPLOYEE_BO,
  supplier: SUPPLIER_BO,
  product: PRODUCT_BO,
  order: ORDER_BO,
  order_detail: ORDER_DETAIL_BO,
  shipper: SHIPPER_BO,
  territory: TERRITORY_BO,
};

// Helper function to get all BOs
export function getNorthwindBOs(): BusinessObjectDefinition[] {
  return Object.values(NORTHWIND_BOS_REGISTRY);
}

// Helper function to get a specific BO
export function getNorthwindBO(key: string): BusinessObjectDefinition | undefined {
  return NORTHWIND_BOS_REGISTRY[key];
}

// Helper function to clone a BO
export function cloneBO(
  sourceBO: BusinessObjectDefinition,
  newName: string,
  newKey: string
): BusinessObjectDefinition {
  return {
    ...sourceBO,
    key: newKey,
    name: newName,
    displayName: `${newName}s`,
    technicalName: newKey,
    isCore: false,
    clonesFrom: sourceBO.key,
    cloneParentKey: sourceBO.key,
    cloneParentDisplayName: sourceBO.name,
    coreFields: [...sourceBO.coreFields],
    customFields: [],
    subtypes: Object.entries(sourceBO.subtypes).reduce(
      (acc, [key, subtype]) => ({
        ...acc,
        [key]: {
          ...subtype,
          isCore: false,
          basedOnEntity: sourceBO.key,
          cloneParentKey: sourceBO.key,
        },
      }),
      {}
    ),
  };
}
