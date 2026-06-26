
const express = require('express');
const router = express.Router();
const Tenant = require('../models/Tenant');

// Create a new tenant
router.post('/', async (req: any, res: any) => {
  const tenant = new Tenant({
    name: req.body.name,
    instance: req.body.instance,
  });

  try {
    const newTenant = await tenant.save();
    res.status(201).json(newTenant);
  } catch (err) {
    res.status(400).json({ message: (err as Error).message });
  }
});

// Other routes (GET, PUT, DELETE) go here

module.exports = router;