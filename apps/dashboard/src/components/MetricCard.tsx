import { Card, Text } from "@mantine/core";

type Props = {
	label: string;
	value: string;
};

export function MetricCard({ label, value }: Props) {
	return (
		<Card className="panel-card" withBorder={true}>
			<Text className="panel-label">{label}</Text>
			<Text className="metric-value">{value}</Text>
		</Card>
	);
}
