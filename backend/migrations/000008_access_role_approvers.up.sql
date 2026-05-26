-- Цепочка согласующих на уровне роли доступа (админ задаёт при создании роли).

CREATE TABLE IF NOT EXISTS access_role_approvers (
    access_role_id UUID NOT NULL REFERENCES access_roles(id) ON DELETE CASCADE,
    step_number INT NOT NULL,
    approver_id UUID NOT NULL REFERENCES users(id),
    PRIMARY KEY (access_role_id, step_number),
    CONSTRAINT access_role_approvers_step_positive CHECK (step_number >= 1)
);

CREATE INDEX IF NOT EXISTS idx_access_role_approvers_approver ON access_role_approvers(approver_id);

-- Для всех существующих ролей — демо-цепочка (те же UUID, что в workflow.DefaultApproverChain).
INSERT INTO access_role_approvers (access_role_id, step_number, approver_id)
SELECT ar.id, s.step, s.approver::uuid
FROM access_roles ar
CROSS JOIN (VALUES
    (1, 'c0000002-0000-0000-0000-000000000002'),
    (2, 'c0000003-0000-0000-0000-000000000003'),
    (3, 'c0000004-0000-0000-0000-000000000004')
) AS s(step, approver)
WHERE NOT EXISTS (SELECT 1 FROM access_role_approvers x WHERE x.access_role_id = ar.id);
