-- +goose Up
-- Feature 033: the simulated GSAP books a purchase request's PO lines as an import of its own source.
ALTER TABLE financial_imports
    DROP CONSTRAINT financial_imports_source_check,
    ADD CONSTRAINT financial_imports_source_check CHECK (source IN ('file', 'demo', 'simulatedGsap'));
