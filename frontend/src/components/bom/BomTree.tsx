import { Link } from 'react-router'
import type { AssemblyNode } from '@/models/items'
import { ROUTE_PATHS } from '@/constants/routes'
import './BomTree.css'

interface BomTreeProps {
  nodes: AssemblyNode[]
}

export function BomTree({ nodes }: BomTreeProps) {
  if (nodes.length === 0) {
    return <p className="bom-tree__empty state">no children</p>
  }

  return (
    <ul className="bom-tree">
      {nodes.map((node, index) => (
        <BomTreeItem key={`${node.id}-${node.position}-${index}`} node={node} />
      ))}
    </ul>
  )
}

function BomTreeItem({ node }: { node: AssemblyNode }) {
  const hasChildren = node.children !== null && node.children.length > 0

  return (
    <li className="bom-tree__node">
      <div className="bom-tree__row">
        <span className="bom-tree__qty">{node.quantity}</span>
        <span className="bom-tree__name">{node.name}</span>
        <span className="bom-tree__sku">{node.sku}</span>
        {node.kind === 'a' && node.version ? (
          <Link className="bom-tree__link" to={ROUTE_PATHS.ASSEMBLY(String(node.id))}>
            v{node.version.version_code}
          </Link>
        ) : null}
      </div>
      {hasChildren && node.children ? (
        <div className="bom-tree__children">
          <BomTree nodes={node.children} />
        </div>
      ) : null}
    </li>
  )
}
