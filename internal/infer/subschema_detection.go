package infer

// DetectSubSchemas has been removed as part of O14 field-agnostic refactor.
// The function assumed a hardcoded discriminator field (e.g., "tipo") to group records,
// which conflicts with field-agnostic design. Schema detection is now driven by
// explicit .stem configuration, not inferred grouping.
