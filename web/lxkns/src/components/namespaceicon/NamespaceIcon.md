At the moment, the Linux kernel defines the following types of namespaces, which
we then adorn using the following icons:

```tsx
import { ComponentCard } from "styleguidist/ComponentCard";
import { NamespaceType } from "models/lxkns";

<div>
  {Object.values(NamespaceType)
    .sort()
    .map((nstype, idx) => (
      <div key={idx}>
        {idx > 0 && <br />}
        <div>{nstype} namespace icon:</div>
        <ComponentCard><NamespaceIcon type={nstype} /></ComponentCard>
      </div>
    ))}
</div>;
```
