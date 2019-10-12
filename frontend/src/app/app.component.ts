import { Component } from '@angular/core';
import { slideInAnimation } from './animations/animations';
import { RouterOutlet } from '@angular/router';

@Component({
  selector: 'app-root',
  templateUrl: './app.component.html',
  styleUrls: ['./app.component.scss'],
  animations: [
    slideInAnimation
  ]
})
export class AppComponent {
  title = 'Communication Demo Center';
  chartClass = 'communication-charts-container';

  // Fetch real data to replace this dummy data later
  donutChartData = [{
    id: 0,
    label: 'voice call',
    value: 40,
    color: 'red',
  }, {
    id: 1,
    label: 'text',
    value: 20,
    color: 'blue',
  }, {
    id: 2,
    label: 'video call',
    value: 30,
    color: 'green',
  }, {
    id: 3,
    label: 'group chat',
    value: 20,
    color: 'yellow',
  }];

  prepareRoute(outlet: RouterOutlet) {
    return outlet && outlet.activatedRouteData && outlet.activatedRouteData['animation'];
  }
}
