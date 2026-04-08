import { ActionIcon, CopyButton, Tooltip } from "@mantine/core";
import { IconCheck, IconCopy } from "@tabler/icons-react";

export function CopyIconButton({
	value,
	label,
}: {
	value: string;
	label: string;
}) {
	return (
		<CopyButton value={value}>
			{({ copied, copy }) => (
				<Tooltip label={copied ? "Copied" : label} withArrow={true}>
					<ActionIcon
						variant="subtle"
						radius="md"
						aria-label={label}
						title={label}
						onClick={copy}
					>
						{copied ? <IconCheck size={16} /> : <IconCopy size={16} />}
					</ActionIcon>
				</Tooltip>
			)}
		</CopyButton>
	);
}
