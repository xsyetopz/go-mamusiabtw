import { Box, Card, Container, Stack, Text, Title } from "@mantine/core";
import type { ReactNode } from "react";

export function AppFrame({ children }: { children: ReactNode }) {
	return (
		<Box className="webui-shell">
			<Container size={1180} px="md" py="xl">
				{children}
			</Container>
		</Box>
	);
}

export function SectionTitle({
	eyebrow,
	title,
	description,
}: {
	eyebrow?: string;
	title: string;
	description?: string;
}) {
	return (
		<Stack gap="xs">
			{eyebrow ? (
				<Text tt="uppercase" fw={700} size="xs" c="brand.6">
					{eyebrow}
				</Text>
			) : null}
			<Title order={2}>{title}</Title>
			{description ? (
				<Text size="sm" c="dimmed" maw={760}>
					{description}
				</Text>
			) : null}
		</Stack>
	);
}

export function SectionCard({
	children,
	className,
}: {
	children: ReactNode;
	className?: string;
}) {
	return (
		<Card withBorder={true} className={className ?? "webui-card"}>
			{children}
		</Card>
	);
}
