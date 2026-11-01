import type { CredentialRequest } from "../lib/use-tools";
import { CREDENTIALS } from "../lib/tools";
import { CredentialForm } from "./credential-form";

interface CredentialDialogProps {
  request: CredentialRequest;
  onClose: () => void;
  onSaved: () => void;
}

export function CredentialDialog({
  request,
  onClose,
  onSaved,
}: CredentialDialogProps) {
  return (
    <dialog open className="modal modal-open">
      <div className="modal-box max-w-2xl bborder bg-base-200 p-4">
        <h3 className="mb-1 text-lg font-bold">
          {CREDENTIALS[request.kind].name} needed
        </h3>
        <p className="mb-4 text-sm opacity-70">{request.reason}</p>
        <CredentialForm
          key={request.kind}
          kind={request.kind}
          submitLabel="Save and retry"
          guideOpen
          onSaved={onSaved}
          onCancel={onClose}
        />
      </div>
    </dialog>
  );
}
