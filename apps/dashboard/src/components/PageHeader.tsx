import { Box, Group, Text, Title } from "@mantine/core";
import type { ReactNode } from "react";

type Props = {
	title: string;
	subtitle: string;
	action?: ReactNode;
};

export function PageHeader({ title, subtitle, action }: Props) {
	return (
		<Group justify="space-between" align="end">
			<Box>
				<Title order={2}>{title}</Title>
				<Text c="dimmed" size="sm">
					{subtitle}
				</Text>
			</Box>
			{action}
		</Group>
	);
}
